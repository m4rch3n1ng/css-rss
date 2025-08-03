package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

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
		http.Error(w, "missing url in query", http.StatusBadRequest)
		return
	}
	url, err := url.Parse(urlQ)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse url %v", urlQ), http.StatusBadRequest)
		return
	}

	selector := query.Get("select")
	if selector == "" {
		http.Error(w, "missing selector", http.StatusBadRequest)
		return
	}

	sel, err := cascadia.Parse(selector)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse selector %s (%s)", selector, err.Error()), http.StatusBadRequest)
		return
	}

	extractor, err := newExtractor(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	excludeQ := query.Get("exclude")
	var exclude cascadia.Matcher
	if excludeQ != "" {
		exclude, err = cascadia.Parse(excludeQ)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse selector %s (%s)", excludeQ, err.Error()), http.StatusBadRequest)
			return
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
		http.Error(w, fmt.Sprintf("failed to parse html (%s)", err.Error()), http.StatusBadRequest)
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

	rss, err := feed.ToAtom()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to convert to rss feed (%s)", err), http.StatusInternalServerError)
	}

	io.WriteString(w, rss)
}

type MatchAttr struct {
	attr    *string
	matcher cascadia.Matcher
}

func newMatchAttr(m string) (*MatchAttr, error) {
	split := strings.SplitN(m, "/", 2)

	var attr string
	if len(split) == 2 {
		attr = split[1]
	}

	matcher, err := cascadia.Parse(split[0])
	if err != nil {
		return nil, err
	}

	return &MatchAttr{attr: &attr, matcher: matcher}, nil
}

type Extractor struct {
	title      cascadia.Matcher
	link       *MatchAttr
	date       cascadia.Matcher
	dateFormat string
}

func newExtractor(query url.Values) (*Extractor, error) {
	titleQuery := query.Get("title")
	if titleQuery == "" {
		return nil, errors.New("title extract query empty")
	}

	title, err := cascadia.Parse(titleQuery)
	if err != nil {
		return nil, err
	}

	linkQuery := query.Get("link")
	var link *MatchAttr
	if linkQuery != "" {
		link, err = newMatchAttr(linkQuery)
		if err != nil {
			return nil, err
		}
	}

	dateQuery := query.Get("date")
	if dateQuery == "" {
		return nil, errors.New("date extract query empty")
	}

	date, err := cascadia.Parse(dateQuery)
	if err != nil {
		return nil, err
	}

	dateFormat := query.Get("dateFormat")
	if dateQuery == "" {
		return nil, errors.New("dateFormat query empty")
	}

	return &Extractor{title, link, date, dateFormat}, nil
}

func matchAttr(n *html.Node, m MatchAttr) *string {
	match := cascadia.Query(n, m.matcher)
	if match == nil {
		return nil
	}

	if m.attr != nil {

		var attr *string
		for _, a := range match.Attr {
			if *m.attr == a.Key {
				attr = &a.Val
				break
			}
		}

		return attr
	} else {
		return &match.FirstChild.Data
	}
}

func extract(node *html.Node, extractor Extractor) *feeds.Item {
	title := cascadia.Query(node, extractor.title)
	if title == nil {
		return nil
	}

	item := feeds.Item{Title: title.FirstChild.Data}

	if extractor.link != nil {
		link := matchAttr(node, *extractor.link)
		if link != nil {
			item.Link = &feeds.Link{Href: *link}
		}
	}

	date := cascadia.Query(node, extractor.date)
	if date != nil {
		date, err := time.Parse(extractor.dateFormat, date.FirstChild.Data)
		if err != nil {
			return nil
		}

		item.Updated = date
		item.Id = date.String()
	}

	return &item
}
