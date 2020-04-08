package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"regexp"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
	"github.com/meilisearch/meilisearch-go"
)

func sha(b []byte) string {
	s := sha1.New()
	_, _ = s.Write(b)
	return fmt.Sprintf("%x", s.Sum(nil))
}

var BASIC_NON_ALPHANUMERIC = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func main() {
	masterKey := flag.String("master-key", "zr", "master key")
	meiliURL := flag.String("meili", "http://127.0.0.1:7700", "meili search url")
	tags := flag.String("tags", "", "tags")
	popularity := flag.Int("popularity", 1, "popularity")
	ptitle := flag.String("title", "", "title")
	pid := flag.String("id", "", "the id of the object, empty means its the sha1 of the content")
	kind := flag.String("kind", "unknown", "kind of object (prependet to the id)")
	flag.Parse()

	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var client = meilisearch.NewClient(meilisearch.Config{
		Host:   *meiliURL,
		APIKey: *masterKey,
	})

	id := *pid
	if id == "" {
		id = sha(in)
	}

	id = fmt.Sprintf("%s_%s", *kind, id)
	id = BASIC_NON_ALPHANUMERIC.ReplaceAllString(id, "_")

	doc := &data.Document{
		Popularity: *popularity,
		Title:      *ptitle,
		Body:       string(in),
		ID:         id,
		Tags:       *tags,
	}

	util.AddAndWait(client, data.IndexName(*kind), []*data.Document{doc})
}
