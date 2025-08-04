package main

import (
	"errors"
	"fmt"
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
		http.ServeFile(w, r, "index.html")
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
		http.Error(w, "missing selector in query", http.StatusBadRequest)
		return
	}

	sel, err := newMultiMatch(selector)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse selector %s (%s)", selector, err.Error()), http.StatusBadRequest)
		return
	}

	extractor, err := newItemExtractor(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	excludeQ := query.Get("exclude")
	var exclude *MultiMatcher
	if excludeQ != "" {
		exclude, err = newMultiMatch(excludeQ)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse selector %s (%s)", excludeQ, err.Error()), http.StatusBadRequest)
			return
		}
	}

	req, err := http.NewRequest(http.MethodGet, urlQ, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (platform; rv:gecko-version) Gecko/gecko-trail Firefox/firefox-version")
	req.Header.Set("Content-Type", "text/html; charset=UTF-8")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if res.StatusCode >= 300 {
		http.Error(w, fmt.Sprintf("%s responded with status %s", url, res.Status), res.StatusCode)
		return
	}

	defer res.Body.Close()
	doc, err := html.Parse(res.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse html (%s)", err.Error()), http.StatusBadRequest)
		return
	}

	title := cascadia.Query(doc, titleSelector)
	if title == nil {
		http.Error(w, "failed to query html <title>", http.StatusBadRequest)
		return
	}

	feed := &feeds.Feed{
		Title: title.FirstChild.Data,
		Link:  &feeds.Link{Href: url.String()},
	}

	q := sel.queryAll(doc)
	for _, n := range q {
		if exclude != nil && len(exclude.queryAll(n)) != 0 {
			continue
		}

		item, err := extractor.extract(n)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		feed.Items = append(feed.Items, item)

		if feed.Updated.Before(item.Updated) {
			feed.Updated = item.Updated
		}
	}

	rss, err := feed.ToAtom()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to convert to rss feed (%s)", err), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(rss))
}

type MultiMatcher []cascadia.Matcher

func newMultiMatch(sel string) (*MultiMatcher, error) {
	sels := strings.Split(sel, ",")

	var m MultiMatcher
	for _, s := range sels {
		mat, err := cascadia.Parse(s)
		if err != nil {
			return nil, err
		}

		m = append(m, mat)
	}

	return &m, nil
}

func (m MultiMatcher) queryAll(doc *html.Node) []*html.Node {
	var nodes []*html.Node
	for _, m := range m {
		n := cascadia.QueryAll(doc, m)
		nodes = append(nodes, n...)
	}

	return nodes
}

type AttrExtractor struct {
	attr    *string
	matcher cascadia.Matcher
}

func newAttrExtractor(m string) (*AttrExtractor, error) {
	split := strings.SplitN(m, "/", 2)

	var attr string
	if len(split) == 2 {
		attr = strings.TrimSpace(split[1])
	}

	matcher, err := cascadia.Parse(split[0])
	if err != nil {
		return nil, err
	}

	return &AttrExtractor{attr: &attr, matcher: matcher}, nil
}

type ItemExtractor struct {
	title      cascadia.Matcher
	link       *AttrExtractor
	date       cascadia.Matcher
	dateFormat string
}

func newItemExtractor(query url.Values) (*ItemExtractor, error) {
	titleQuery := query.Get("title")
	if titleQuery == "" {
		return nil, errors.New("missing selector for title in query")
	}

	dateQuery := query.Get("date")
	if dateQuery == "" {
		return nil, errors.New("missing selector for date in query")
	}

	dateFormat := query.Get("dateFormat")
	if dateFormat == "" {
		return nil, errors.New("missing dateFormat in query")
	}

	title, err := cascadia.Parse(titleQuery)
	if err != nil {
		return nil, err
	}

	linkQuery := query.Get("link")
	var link *AttrExtractor
	if linkQuery != "" {
		link, err = newAttrExtractor(linkQuery)
		if err != nil {
			return nil, err
		}
	}

	date, err := cascadia.Parse(dateQuery)
	if err != nil {
		return nil, err
	}

	return &ItemExtractor{title, link, date, dateFormat}, nil
}

func (m AttrExtractor) extractAttr(n *html.Node) *string {
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

func (e ItemExtractor) extract(n *html.Node) (*feeds.Item, error) {
	title := cascadia.Query(n, e.title)
	if title == nil {
		return nil, errors.New("failed to query the title")
	}

	item := feeds.Item{Title: title.FirstChild.Data}

	if e.link != nil {
		link := e.link.extractAttr(n)
		if link != nil {
			item.Link = &feeds.Link{Href: *link}
		}
	}

	date := cascadia.Query(n, e.date)
	if date != nil {
		date, err := time.Parse(e.dateFormat, date.FirstChild.Data)
		if err != nil {
			return nil, err
		}

		item.Updated = date
		item.Id = date.String()
	}

	return &item, nil
}
