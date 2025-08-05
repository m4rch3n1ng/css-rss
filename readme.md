<p align="center">create an rss feed via css selectors</p>

you can run the http server via

```
$ go run .
```

or you can deploy via `docker compose up --build -d`.

to create a new feed, navigate to the webpage. there you will be presented by website with input elements, where you can add your desired parameters. pressing the create button navigates to a new website with the parameters in the url. this is the url of the created rss feed.  
as of writing this, there is an instance hosted at https://css-rss.m4rch.xyz, though no promises to the future of this.

an example url will look something like this: `https://css-rss.m4rch.xyz/?url=https%3A%2F%2Fmydramalist.com%2F789950-not-friend%2Fepisodes&select=.episodes+%3E+div&title=.title+%3E+a&date=.air-date&dateFormat=Jan+02%2C+2006&link=.title+%3E+a%2Fhref&exclude=.cover.missing`
