package main

import (
	"os"
	"log"
	"fmt"
	"time"
	"sort"
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"math/rand"
	"html/template"
	"golang.org/x/net/html"
	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/extensions"
	"github.com/glacjay/goini"
)

var global struct {
	max_posts_per_author int
	max_posts_per_page   int
	template             string
	output               string
}

func post_to_hash(post map[string]interface{}) (hash string) {
	hash = post["published"].(time.Time).Format("20060102")
}

func keys_from_map(in map[string]string) []string {
	keys := make([]string, len(in))
	i := 0
	for k := range in {
		keys[i] = k
		i++
	}
	return keys
}

func add_extensions(post *map[string]interface{}, main string, extensions map[string][]ext.Extension) {
	for name, exts := range extensions {
		for _, ext := range exts {
			if len(ext.Value) > 0 {
//				log.Printf("### ADDING %s:%s = %+v", main, name, ext.Value)
				(*post)[main + "_" + name] = ext.Value
			}
			for a, b := range ext.Attrs {
				(*post)[main + "_" + name + "__" + a] = b
			}
			add_extensions(post, main + "__" + name, ext.Children)
		}
	}
}

var debug = false

func get_first_image(s string) string {
	re := regexp.MustCompile(`<img.*?src *= *["']?(.*?)["' >]`)
	match := re.FindStringSubmatch(s)
	if len(match) > 1 {
		return match[1]
	} else {
		return ""
	}
}

func get_meta(node *html.Node, name string) (r string) {
	if node==nil {
		return ""
	}
	if node.Type==html.ElementNode && node.Data=="meta" {
		var this_name, this_content string
			for _, e := range node.Attr {
				if e.Key=="name" {
					this_name = e.Val
				}
				if e.Key=="content" {
					this_content = e.Val
				}
			}
		if name==this_name {
			return this_content
		}
	}
	r = get_meta(node.FirstChild, name)
		if r != "" {
			return r
		}
	r = get_meta(node.NextSibling, name)
		return r
}


func clean_html(s string) string {
	root, err := html.Parse(strings.NewReader(s))
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return ""
	}
	var b bytes.Buffer
	html.Render(&b, root)
	s = b.String()

	re := regexp.MustCompile("^.*?<body[^>]*>")
	s = re.ReplaceAllString(s, "")
	re = regexp.MustCompile("</body>.*$")
	s = re.ReplaceAllString(s, "")
	return s
}

var i18n_dows map[string][]string
var i18n_months map[string][]string

func init() {
	i18n_dows = make(map[string][]string)
	i18n_months = make(map[string][]string)
	i18n_dows["spanish"] = []string{
		"domingo",
		"lunes",
		"martes",
		"miércoles",
		"jueves",
		"viernes",
		"sábado",
	}
	i18n_months["spanish"] = []string{
		"enero",
		"febrero",
		"marzo",
		"abril",
		"mayo",
		"junio",
		"julio",
		"agosto",
		"septiembre",
		"octubre",
		"noviembre",
		"diciembre",
	}
}

func main() {
	config_file := "/etc/planet.ini"
	if len(os.Args)>1 && os.Args[1]=="-d" {
		debug = true
		os.Args = os.Args[1:]
	}
	if len(os.Args)==3 && os.Args[1]=="-c" {
		config_file = os.Args[2]
	}
	vars := make(map[string]interface{})
	posts := make([]map[string]interface{}, 0, 200)
	blogs := make([]map[string]interface{}, 0, 200)
	config, err := ini.Load(config_file)
	if err != nil {
		log.Println(err)
		return
	}
	global.max_posts_per_author, _ = strconv.Atoi(config["_global"]["max_posts_per_author"])
	global.max_posts_per_page, _ = strconv.Atoi(config["_global"]["max_posts_per_page"])
	global.template = config["_global"]["template"]
	global.output = config["_global"]["output"]
//	x := 0
	for name, content := range config {
//		x++
//		if x > 5 {
//			break
//		}
		if content["rss"] != "" {
			blog := make(map[string]interface{})
			log.Printf("Reading feed %s...", content["rss"])
			fp := gofeed.NewParser()
			feed, err := fp.ParseURL(content["rss"])
			if err != nil {
				log.Printf("Error reading feed %s: %v", name, err)
			}
			blog["name"] = feed.Title
			blog["url"] = feed.Link
			if len(feed.Items) > 1 {
				blog["time"] = *(feed.Items[0].PublishedParsed)
			} else {
				var a time.Time
				blog["time"] = a
			}
			blogs = append(blogs, blog)
//			log.Printf("[%s] %s (%d items)", name, feed.Title, len(feed.Items))
			if debug {
				log.Printf("### feed = %v", feed)
			}
			i := 0
			for _, item := range feed.Items {
				if i >= global.max_posts_per_author {
					break
				}
				post := make(map[string]interface{})
				post["origin"] = name
				post["id"] = item.GUID
				post["published"] = *item.PublishedParsed
				post["title"] = item.Title
				post["description"] = item.Description
				post["content"] = clean_html(item.Content)
				post["link"] = item.Link
				if item.Author != nil {
					post["author_name"] = item.Author.Name
					post["author_email"] = item.Author.Email
				}
				if item.Image != nil {
					post["image"] = item.Image.URL
				} else {
					post["image"] = get_first_image(item.Content)
				}
				post["feed_title"] = feed.Title
				post["feed_link"] = feed.Link
				post["feed_description"] = feed.Description
				if content["title"] != "" {
					post["blog_title"] = content["title"]
				} else {
					post["blog_title"] = feed.Title
				}
				if content["description"] != "" {
					post["blog_description"] = content["description"]
				} else if feed.Description != "" {
					post["blog_description"] = feed.Description
				} else {
					post["blog_description"] = feed.Title
				}
				if content["avatar"] != "" {
					post["blog_avatar"] = content["avatar"]
				} else {
					post["blog_avatar"] = config["_global"]["default_avatar"]
				}
				if content["url"] != "" {
					post["blog_url"] = content["url"]
				} else {
					post["blog_url"] = feed.Link
				}
				// TODO: add some more fields...
				// TODO: add extensions
				for extmain, rest := range item.Extensions {
					add_extensions(&post, extmain, rest)
				}
				posts = append(posts, post)
//				log.Printf("..(%s) %s", item.PublishedParsed.Format("2006-01-02"), item.Title)
				i++
			}
		}
//		log.Println("[" + name + "] " + fmt.Sprint(keys_from_map(content)))
	}
	sort.Slice(posts, func(i, j int) bool { return posts[j]["published"].(time.Time).Before(posts[i]["published"].(time.Time)) })
	sort.Slice(blogs, func(i, j int) bool { return blogs[j]["time"].(time.Time).Before(blogs[i]["time"].(time.Time)) })
	if len(posts) > global.max_posts_per_page {
		posts = posts[:global.max_posts_per_page]
	}
	for i, _ := range posts {
		posts[i]["index"] = i
	}
	vars["config"] = config
	vars["posts"] = posts
	vars["blogs"] = blogs
	vars["last_update"] = time.Now()
	for x, y := range config["_global"] {
		vars[x] = y
	}
	funcMap := template.FuncMap{
                "noescape": func(s string) template.HTML {
                        return template.HTML(s)
                },
                "add": func(a, b int) int {
                        return a + b
                },
                "sub": func(a, b int) int {
                        return a - b
                },
                "mul": func(a, b int) int {
                        return a * b
                },
                "div": func(a, b int) int {
                        return a / b
                },
                "mod": func(a, b int) int {
                        return a % b
                },
                "rand": rand.Float64,
                "html2text": func(s string) string {
			re := regexp.MustCompile("<!--.*?-->")
			s = re.ReplaceAllString(s, " ")
			re = regexp.MustCompile("<[^>]*>")
			return re.ReplaceAllString(s, " ")
                },
                "truncate": func(size int, s string) string {
			l := len(s)
			if size > l {
				size = l
			}
			return s[:size]
                },
                "longdate": func(lang string, t time.Time) string {
			dow := strings.Title(i18n_dows[lang][t.Weekday()])
			day := t.Day()
			month := i18n_months[lang][t.Month()-1]
			year := t.Year()
			return fmt.Sprintf("%s, %d de %s de %d", dow, day, month, year)
                },
                "shortdate": func(lang string, t time.Time) string {
			return t.Format("02-01-2006")
                },
                "hhmm": func(t time.Time) string {
			return t.Format("15:04")
                },
        }
	t, err := template.New(global.template).Funcs(funcMap).ParseFiles(global.template)
	if err != nil {
		log.Print(err)
		return
	}
	f, err := os.Create(global.output)
	if err != nil {
		log.Println("create file: ", err)
		return
	}
	defer f.Close()
	if debug {
		log.Printf("### vars = %+v", vars)
	}
	err = t.Execute(f, vars)
	if err != nil {
		log.Println("execute: ", err)
		return
	}
}
