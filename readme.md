
<p align="center">create an rss feed via css selectors</p>

you can run the http server via

```
$ go run .
```

to create an rss feed you (currently) have to manually edit the url to this scheme:

```
http://<instance-of-css-rss>/?url=<url-to-convert>&select=<css-selector-for-item>&title=<css-selector-for-title>&date=<css-selector-for-date>&dateFormat=<go-date-format>&link=<css-selector-for-link>/<attr-for-link>
```

all of these parameters are mandatory. you can additionally specify `exclude` parameter.

an example url will look something like this: `https://css-rss.m4rch.xyz/?url=https://mydramalist.com/789950-not-friend/episodes&select=.episodes%20%3E%20div&title=.title%20%3E%20a&date=.air-date&dateFormat=Jan%2002,%202006&exclude=.cover.missing&link=.title%3Ea/href`
