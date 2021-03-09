package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	"cgt.name/pkg/go-mwclient"
	"github.com/lightstep/otel-launcher-go/launcher"
	"github.com/pitr/gig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

var (
	version = "unknown"
	tracer  trace.Tracer
)

func main() {
	ls := launcher.ConfigureOpentelemetry(
		launcher.WithServiceName("wp"),
		launcher.WithMetricsEnabled(true),
		launcher.WithServiceVersion(version),
		launcher.WithAccessToken(os.Getenv("LS_TOKEN")),
	)
	defer ls.Shutdown()
	tracer = otel.Tracer("wp")

	// preload main client
	_, err := getClient("en")
	if err != nil {
		panic(err)
	}

	g := gig.New()

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

	g.Use(gig.Logger(), func(f gig.HandlerFunc) gig.HandlerFunc {
		return func(c gig.Context) error {
			opts := trace.WithSpanKind(trace.SpanKindServer)
			ctx, span := tracer.Start(context.Background(), c.Path(), opts)
			span.SetAttributes(semconv.HTTPURLKey.String(c.RequestURI()))
			c.Set("ctx", ctx)
			defer span.End()

			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}

					stack := make([]byte, 4<<10)
					length := runtime.Stack(stack, true)
					fmt.Printf("[PANIC RECOVER] %v %s\n", err, stack[:length])

					c.Error(err)
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				}
			}()
			err := f(c)

			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetAttributes(semconv.HTTPStatusCodeKey.Int(int(c.Response().Status)))
				span.SetAttributes(semconv.HTTPResponseContentLengthKey.Int64(int64(c.Response().Size)))
			}

			return err
		}
	})

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
		lang = c.Param("lang")
		name = c.Param("*")
	)

	ctx := c.Get("ctx").(context.Context)

	_, sp := tracer.Start(ctx, "getClient")

	wp, err := getClient(lang)
	sp.End()

	if err != nil {
		return err
	}

	_, sp = tracer.Start(ctx, "getPageByName")
	page, _, err := wp.GetPageByName(name)
	sp.End()

	if err == mwclient.ErrPageNotFound {
		return c.NoContent(gig.StatusNotFound, err.Error())
	}
	if err != nil {
		return err
	}

	_, sp = tracer.Start(ctx, "convert")
	body := convert(page)
	sp.End()

	return c.Render("show", &showWrapper{
		Lang:  lang,
		Title: strings.ReplaceAll(name, "_", " "),
		Body:  body,
	})
}
