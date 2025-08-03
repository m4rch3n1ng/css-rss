package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/andybalholm/cascadia"
	"github.com/gorilla/feeds"
	"golang.org/x/net/html"
)

func main() {
	http.HandleFunc("/", handleRequest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var titleSelector cascadia.Matcher = cascadia.MustCompile("title")

func handleRequest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if len(query) == 0 {
		io.WriteString(w, "yay")
		return
	}

	urlQ := query.Get("url")
	if urlQ == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}
	url, err := url.Parse(urlQ)
	if err != nil {
		log.Fatal(err)
	}

	selector := query.Get("select")
	if selector == "" {
		http.Error(w, "missing selector", http.StatusBadRequest)
		return
	}

	sel, err := cascadia.Parse(selector)
	if err != nil {
		log.Fatal(err)
	}

	extractor, err := newExtractor(query)
	if err != nil {
		log.Fatal(err)
	}

	excludeQ := query.Get("exclude")
	var exclude cascadia.Matcher
	if excludeQ != "" {
		exclude, err = cascadia.Parse(excludeQ)
		if err != nil {
			log.Fatal(err)
		}
	}

	req, err := http.NewRequest(http.MethodGet, urlQ, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (platform; rv:gecko-version) Gecko/gecko-trail Firefox/firefox-version")
	req.Header.Set("Content-Type", "text/html; charset=UTF-8")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer res.Body.Close()
	doc, err := html.Parse(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	title := cascadia.Query(doc, titleSelector).FirstChild.Data
	feed := &feeds.Feed{
		Title: title,
		Link:  &feeds.Link{Href: fmt.Sprintf("%s://%s", url.Scheme, url.Host)},
	}

	q := cascadia.QueryAll(doc, sel)
	for _, n := range q {
		if exclude != nil && len(cascadia.QueryAll(n, exclude)) != 0 {
			continue
		}

		item := extract(n, *extractor)
		if item != nil {
			feed.Items = append(feed.Items, item)
		}
	}

	rss, err := feed.ToRss()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, "yo, %v", rss)
}

type Extractor struct {
	title cascadia.Matcher
	link  cascadia.Matcher
	date  cascadia.Matcher
}

func newExtractor(query url.Values) (*Extractor, error) {
	titleQuery := query.Get("title")
	if titleQuery == "" {
		log.Fatal("title extract query empty")
	}

	title, err := cascadia.Parse(titleQuery)
	if err != nil {
		return nil, err
	}

	linkQuery := query.Get("link")
	var link cascadia.Matcher
	if linkQuery != "" {
		link, err = cascadia.Parse(linkQuery)
		if err != nil {
			return nil, err
		}
	}

	dateQuery := query.Get("date")
	var date cascadia.Matcher
	if dateQuery != "" {
		date, err = cascadia.Parse(dateQuery)
		if err != nil {
			return nil, err
		}
	}

	return &Extractor{title, link, date}, nil
}

func extract(node *html.Node, extractor Extractor) *feeds.Item {
	title := cascadia.Query(node, extractor.title)
	if title == nil {
		return nil
	}

	item := feeds.Item{Title: title.FirstChild.Data}

	if extractor.link != nil {
		link := cascadia.Query(node, extractor.link)
		if link != nil {
			item.Link = &feeds.Link{Href: link.FirstChild.Data}
		}
	}

	if extractor.date != nil {
		date := cascadia.Query(node, extractor.date)
		if date != nil {
			item.Description = date.FirstChild.Data
		}
	}

	return &item
}
