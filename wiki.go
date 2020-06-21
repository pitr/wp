package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"cgt.name/pkg/go-mwclient"
	"github.com/antonholmquist/jason"
	"github.com/d4l3k/wikigopher/wikitext"
	"golang.org/x/net/html"
)

var (
	wpClients        = map[string]*mwclient.Client{}
	badLanguageError = errors.New("language must be exactly 2 letters")
)

func getClient(lang string) (c *mwclient.Client, err error) {
	var ok bool
	if len(lang) != 2 {
		return nil, badLanguageError
	}
	if c, ok = wpClients[lang]; ok {
		return
	}
	c, err = mwclient.New(fmt.Sprintf("https://%s.wikipedia.org/w/api.php", lang), "gemini-proxy")
	if err != nil {
		return
	}
	wpClients[lang] = c
	return
}

type searchResult struct {
	Name string
	Path string
}

func search(q string) ([]searchResult, error) {
	c, err := getClient("en")
	if err != nil {
		return nil, err
	}
	resp, err := c.GetRaw(map[string]string{
		"action": "opensearch",
		"search": q,
	})
	if err != nil {
		return nil, err
	}
	val, err := jason.NewValueFromBytes(resp)
	if err != nil {
		return nil, err
	}
	arr, err := val.Array()
	if err != nil {
		return nil, err
	}
	arr, err = arr[1].Array()
	if err != nil {
		return nil, err
	}
	var results []searchResult
	for _, el := range arr {
		s, err := el.String()
		if err != nil {
			return nil, err
		}
		results = append(results, searchResult{
			Name: s,
			Path: strings.ReplaceAll(s, " ", "_"),
		})
	}
	return results, nil
}

func convert(in string) string {
	text := []byte(in)
	v, err := wikitext.Parse(
		"file.wikitext",
		[]byte(append(text, '\n')),
		wikitext.GlobalStore("len", len(text)),
		wikitext.GlobalStore("text", text),
		wikitext.GlobalStore("opts", nil),
		wikitext.Recover(false))
	if err != nil {
		println(err)
		return "could not parse: " + err.Error()
	}

	var doc *html.Node

	for doc == nil && v != nil {
		switch val := v.(type) {
		case *html.Node:
			doc = val
		default:
			println(v)
		}
	}

	if doc == nil {
		return "Could not render page"
	}

	var buf strings.Builder
	f := newFooter()
	render(&buf, f, doc)
	return buf.String() + f.String()
}

func render(buf *strings.Builder, footer *footer, node *html.Node) {
	for {
		if node == nil {
			return
		}
		switch node.Type {
		case html.ErrorNode:
		case html.CommentNode:
		case html.DoctypeNode:
		case html.TextNode:
			buf.WriteString(strings.ReplaceAll(node.Data, "\n", ""))
		case html.DocumentNode:
			render(buf, footer, node.FirstChild)
		case html.ElementNode:
			switch node.Data {
			case "h1":
				buf.WriteString(footer.String())
				footer.Reset()
				var t strings.Builder
				getText(&t, node)
				name := t.String()
				buf.WriteString(fmt.Sprintf("# %s\n", name))
			case "h2":
				buf.WriteString(footer.String())
				footer.Reset()
				var t strings.Builder
				getText(&t, node)
				name := t.String()
				buf.WriteString(fmt.Sprintf("## %s\n", name))
			case "h3":
				buf.WriteString(footer.String())
				footer.Reset()
				var t strings.Builder
				getText(&t, node)
				name := t.String()
				buf.WriteString(fmt.Sprintf("### %s\n", name))
			case "h4":
				buf.WriteString(footer.String())
				footer.Reset()
				var t strings.Builder
				getText(&t, node)
				name := t.String()
				buf.WriteString(fmt.Sprintf("### %s\n", name))
			case "h5":
				buf.WriteString(footer.String())
				footer.Reset()
				var t strings.Builder
				getText(&t, node)
				name := t.String()
				buf.WriteString(fmt.Sprintf("### %s\n", name))
			case "h6":
				buf.WriteString(footer.String())
				footer.Reset()
				var t strings.Builder
				getText(&t, node)
				name := t.String()
				buf.WriteString(fmt.Sprintf("### %s\n", name))
			case "li":
				var t strings.Builder
				render(&t, footer, node.FirstChild)
				item := t.String()
				if item != "" && item != "." {
					buf.WriteString(fmt.Sprintf("* %s\n", item))
				}
			case "p":
				if node.FirstChild != nil && (node.FirstChild.Type != html.TextNode || node.FirstChild.Data != "") {
					buf.WriteString("\n\n")
					render(buf, footer, node.FirstChild)
					buf.WriteString("\n")
					buf.WriteString(footer.String())
					footer.Reset()
				}
			case "a":
				var href string
				for _, a := range node.Attr {
					if a.Key == "href" {
						href = a.Val
						break
					}
				}
				if href != "" {
					var t strings.Builder
					getText(&t, node)
					name := t.String()
					footer.addLink(name, href)
					buf.WriteString(name)
				} else {
					render(buf, footer, node.FirstChild)
				}
			case "b":
				buf.WriteString("*")
			case "ref":
				for {
					node = node.NextSibling
					if node == nil {
						break
					}
					if node.Type == html.ElementNode && node.Data == "ref" {
						break
					}
				}
			default:
			}
		default:
			buf.WriteString("unknown\n")
		}
		if node == nil {
			break
		}
		node = node.NextSibling
	}
}

func getText(buf *strings.Builder, node *html.Node) {
	if node == nil {
		return
	}
	switch node.Type {
	case html.TextNode:
		buf.WriteString(strings.ReplaceAll(node.Data, "\n", ""))
	case html.ElementNode:
		child := node.FirstChild
		for {
			if child == nil {
				break
			}
			getText(buf, child)
			child = child.NextSibling
		}
	}
}

type footer struct {
	buf strings.Builder
}

func newFooter() *footer {
	return &footer{}
}

func (f *footer) addLink(name, href string) {
	href = url.PathEscape(href)
	f.buf.WriteString(fmt.Sprintf("=> %s %s\n", href, name))
}

func (f *footer) String() string {
	return "\n" + f.buf.String()
}

func (f *footer) Reset() {
	f.buf.Reset()
}
