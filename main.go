package main

import (
	"strings"

	"github.com/pitr/gig"
)

func main() {
	// preload main client
	_, err := getClient("en")
	if err != nil {
		panic(err)
	}

	g := gig.Default()
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

func handleRobot(c gig.Context) error {
	// otherwise crawler index would explode, also unnecessary traffic
	return c.Text("User-agent: *\nAllow: /$\nDisallow: /\n")
}

func handleHome(c gig.Context) error {
	return c.Render("index", nil)
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

func handleShow(c gig.Context) error {
	var (
		lang = c.Param("lang")
		name = c.Param("*")
	)

	wp, err := getClient(lang)
	if err != nil {
		return err
	}

	page, _, err := wp.GetPageByName(name)
	if err != nil {
		return err
	}

	return c.Render("show", &showWrapper{
		Lang:  lang,
		Title: strings.ReplaceAll(name, "_", " "),
		Body:  convert(page),
	})
}
