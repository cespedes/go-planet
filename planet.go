package main

import (
	"os"
	"fmt"
	"log"
	"time"
	"sort"
	"strconv"
	"html/template"
	"github.com/mmcdole/gofeed"
	"github.com/glacjay/goini"
)

var global struct {
	max_posts_per_author int
	max_posts_per_page   int
	template             string
	output               string
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

type entry struct {
	Origin    string
	Published time.Time
	Title     string
	Item      *gofeed.Item
}

func (e entry) String() string {
	return fmt.Sprintf("%s [%s] %s", e.Published.Format("2006-01-02"), e.Origin, e.Title)
}

func main() {
	config_file := "planet.ini"
	if len(os.Args)==3 && os.Args[1]=="-c" {
		config_file = os.Args[2]
	}
	vars := make(map[string]interface{})
	entries := make([]entry, 0, 200)
	config, err := ini.Load(config_file)
	if err != nil {
		log.Println(err)
		return
	}
	global.max_posts_per_author, _ = strconv.Atoi(config["_global"]["max_posts_per_author"])
	global.max_posts_per_page, _ = strconv.Atoi(config["_global"]["max_posts_per_page"])
	global.template = config["_global"]["template"]
	global.output = config["_global"]["output"]
	x := 0
	for name, content := range config {
		x++
		if x > 5 {
			break
		}
		if content["rss"] != "" {
			fp := gofeed.NewParser()
			feed, _ := fp.ParseURL(content["rss"])
			log.Printf("[%s] %s (%d items)", name, feed.Title, len(feed.Items))
			log.Printf("### %v", feed)
			i := 0
			for _, item := range feed.Items {
				if i >= global.max_posts_per_author {
					break
				}
				entries = append(entries, entry{Origin:name, Published:*item.PublishedParsed, Title:item.Title, Item:item})
//				log.Printf("..(%s) %s", item.PublishedParsed.Format("2006-01-02"), item.Title)
				i++
			}
		}
//		log.Println("[" + name + "] " + fmt.Sprint(keys_from_map(content)))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[j].Published.Before(entries[i].Published) })
	if len(entries) > global.max_posts_per_page {
		entries = entries[:global.max_posts_per_page]
	}
	for _, entry := range entries {
		log.Println(entry)
	}
	vars["config"] = config
	vars["posts"] = entries
	for x, y := range config["_global"] {
		vars[x] = y
	}
	t, err := template.ParseFiles(global.template)
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
	log.Printf("%+v",vars)
	err = t.Execute(f, vars)
	if err != nil {
		log.Println("execute: ", err)
		return
	}

}
