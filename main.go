package main

import (
	"crypto/tls"
	"io/ioutil"
	"strings"
	"time"

	"cgt.name/pkg/go-mwclient"
	"github.com/pitr/gig"
)

func main() {
	// preload main client
	_, err := getClient("en")
	if err != nil {
		panic(err)
	}

	g := gig.Default()

	g.TLSConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if !strings.Contains(hello.ServerName, ".glv.one") {
			return nil, nil
		}
		c, err := ioutil.ReadFile("/meta/credentials/letsencrypt/current/fullchain.pem")
		if err != nil {
			return nil, err
		}
		k, err := ioutil.ReadFile("/meta/credentials/letsencrypt/current/privkey.pem")
		if err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair(c, k)
		if err != nil {
			return nil, err
		}
		return &cert, nil
	}
	g.Renderer = &Template{}

	g.Handle("/", handleHome)
	g.Handle("/search", func(c gig.Context) error {
		// redirect old search path
		return c.NoContent(gig.StatusRedirectPermanent, "/en/")
	})
	g.Handle("/robots.txt", handleRobot)
	g.Handle("/:lang/", handleSearch)
	g.Handle("/:lang/*", handleShow)

	panic(g.Run("wp.crt", "wp.key"))
}

var langs = []string{"en", "ar", "de", "es", "fr", "it", "nl", "ja", "pl", "pt", "ru", "sv", "uk", "vi", "zh", "id", "ms", "zh", "bg", "ca", "cs", "da", "eo", "eu", "fa", "he", "ko", "hu", "no", "ro", "sr", "sh", "fi", "tr", "ast", "bs", "et", "el", "simple", "gl", "hr", "lv", "lt", "ml", "nn", "sk", "sl", "th"}

func handleRobot(c gig.Context) error {
	// otherwise crawler index would explode, also unnecessary traffic
	return c.Text("User-agent: *\nDisallow: /search\nDisallow: /%s", strings.Join(langs, "\nDisallow: /"))
}

func handleHome(c gig.Context) error {
	return c.Render("index", struct{ Old bool }{strings.Contains(c.RequestURI(), "wp.pitr.ca")})
}

type searchResultWrapper struct {
	Query, Lang string
	Result      []searchResult
}

func handleSearch(c gig.Context) error {
	lang := c.Param("lang")

	q, err := c.QueryString()
	if err != nil {
		return c.NoContent(gig.StatusInput, "Invalid search query, try again")
	}
	if len(q) == 0 {
		return c.NoContent(gig.StatusInput, "Enter search query")
	}

	result, err := search(lang, q)
	if err != nil {
		println(err.Error())
		return gig.ErrPermanentFailure
	}
	return c.Render("search", &searchResultWrapper{
		Query:  q,
		Lang:   lang,
		Result: result,
	})
}

type showWrapper struct {
	Title, Body, Lang string
}

func measure(m string, f func()) {
	t := time.Now()
	f()
	println(m, time.Since(t))
}

func handleShow(c gig.Context) error {
	var (
		lang       = c.Param("lang")
		name       = c.Param("*")
		wp         *mwclient.Client
		err        error
		page, body string
	)

	measure("getClient", func() {
		wp, err = getClient(lang)
	})
	if err != nil {
		return err
	}

	measure("GetPageByName", func() {
		page, _, err = wp.GetPageByName(name)
	})
	if err == mwclient.ErrPageNotFound {
		return c.NoContent(gig.StatusNotFound, err.Error())
	}
	if err != nil {
		return err
	}

	measure("convert", func() {
		body = convert(page)
	})

	return c.Render("show", &showWrapper{
		Lang:  lang,
		Title: strings.ReplaceAll(name, "_", " "),
		Body:  body,
	})
}
