package lib

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

var (
	whitespace = regexp.MustCompile(`\s+`)
)

func ExtractImageURL(n *html.Node) string {
	if url := extractOpengraphImage(n); url != "" {
		return url
	}
	if url := extractTwitterImage(n); url != "" {
		return url
	}
	return ""
}

func extractOpengraphImage(n *html.Node) string {
	elem := htmlquery.FindOne(n, "//meta[@property = 'og:image']")
	if elem != nil {
		for _, attr := range elem.Attr {
			if attr.Key == "content" {
				return attr.Val
			}
		}
	}
	return ""
}

func extractTwitterImage(n *html.Node) string {
	elem := htmlquery.FindOne(n, "//meta[@name = 'twitter:image']")
	if elem != nil {
		for _, attr := range elem.Attr {
			if attr.Key == "content" {
				return attr.Val
			}
		}
	}
	return ""
}

func SelectText(n *html.Node, xpath string) string {
	node := htmlquery.FindOne(n, xpath)
	return digForText(node)
}

func digForText(n *html.Node) string {
	if n == nil {
		return ""
	}
	buf := new(bytes.Buffer)
	dig(n, buf)
	return compactWhitespace(buf.String())
}

func dig(n *html.Node, buf *bytes.Buffer) {
	if n == nil {
		return
	}
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		dig(c, buf)
	}
}

func compactWhitespace(s string) string {
	s = whitespace.ReplaceAllString(s, " ")
	s = strings.Trim(s, " ")
	return s
}
