package models

import (
	"crypto/sha1"
	"fmt"
)

type EndpointContent struct {
	Text     string
	Title    string
	ImageURL string
}

func DigestContent(content string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(content)))
}
