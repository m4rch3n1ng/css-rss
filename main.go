package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

func main() {
	http.HandleFunc("/", handleRequest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if len(query) == 0 {
		io.WriteString(w, "yay")
		return
	}

	url := query.Get("url")
	if url == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
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

	extractQ := query.Get("extract")
	var extract cascadia.Matcher
	if extractQ != "" {
		extract, err = cascadia.Parse(extractQ)
		if err != nil {
			log.Fatal(err)
		}
	}

	excludeQ := query.Get("exclude")
	var exclude cascadia.Matcher
	if excludeQ != "" {
		exclude, err = cascadia.Parse(excludeQ)
		if err != nil {
			log.Fatal(err)
		}
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
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

	q := cascadia.QueryAll(doc, sel)

	var m []string
	for _, el := range q {
		if exclude != nil && len(cascadia.QueryAll(el, exclude)) != 0 {
			continue
		}

		var data string
		if extract != nil {
			data = cascadia.Query(el, extract).FirstChild.Data
		} else {
			data = el.FirstChild.Data
		}

		m = append(m, data)
	}

	fmt.Fprintf(w, "yo, %v", m)
}
