
<p align="center">create an rss feed via css selectors</p>

you can run the http server via

```
$ go run .
```

or you can deploy via `docker compose up --build -d`.

to create a new feed, navigate to the webpage. there you will be presented by a very modern and ✨fancy✨ website with input elements, where you can add your desired parameters. pressing the create button navigates to a new website with the parameters in the url.  
as of writing this, there is an instance hosted at https://css-rss.m4rch.xyz, though no promises to the future of this.

an example url will look something like this: `https://css-rss.m4rch.xyz/?url=https://mydramalist.com/789950-not-friend/episodes&select=.episodes%20%3E%20div&title=.title%20%3E%20a&date=.air-date&dateFormat=Jan%2002,%202006&exclude=.cover.missing&link=.title%3Ea/href`
