package data

import "fmt"

func IndexName(k string) string {
	return fmt.Sprintf("zr_x2_%s", k)
}

type Document struct {
	ID string `json:"id"`

	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Tags  string `json:"tags,omitempty"`

	Popularity int `json:"popularity,omitempty"`
}
