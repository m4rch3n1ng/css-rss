
<p align="center">create an rss feed via css selectors</p>

you can run the http server via

```
$ go run .
```

to create an rss feed you (currently) have to manually edit the url to this scheme:

```
http://<instance-of-css-rss>/?url=<url-to-convert>&select=<css-selector-for-item>&title=<css-selector-for-title>
```

all of these parameters are mandatory. you can additionally specify `date` and `exclude` parameters.
