package main

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	whitespace = regexp.MustCompile(`\s+`)
)

func collectText(n *html.Node) string {
	buf := new(bytes.Buffer)
	recursiveCollect(n, buf)
	return compactWhitespace(buf.String())
}

func recursiveCollect(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		recursiveCollect(c, buf)
	}
}

func compactWhitespace(s string) string {
	s = whitespace.ReplaceAllString(s, " ")
	s = strings.Trim(s, " ")
	return s
}
