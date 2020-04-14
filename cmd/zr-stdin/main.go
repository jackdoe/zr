package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	_ "net/http/pprof"
	"os"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
)

func sha(b []byte) string {
	s := sha1.New()
	_, _ = s.Write(b)
	return fmt.Sprintf("%x", s.Sum(nil))
}

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "root")
	kind := flag.String("k", "unknown", "kind of object (prependet to the id)")
	tags := flag.String("tags", "", "tags")
	popularity := flag.Int("popularity", 1, "popularity")
	ptitle := flag.String("title", "", "title")
	fn := flag.String("file", "", "filename or empty for stdin")
	pid := flag.String("id", "", "the id of the object, empty means its the sha1 of the content")
	flag.Parse()

	var in []byte
	var err error
	if *fn == "" {
		in, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
	} else {
		in, err = ioutil.ReadFile(*fn)
		if err != nil {
			panic(err)
		}
	}

	if *root == "" {
		log.Fatal("root")
	}

	if *kind == "" {
		log.Fatal("kind")
	}

	store := data.NewStore(*root, *kind)
	defer store.Close()

	id := *pid
	if id == "" {
		id = sha(in)
	}

	id = fmt.Sprintf("%s_%s", *kind, id)

	doc := &data.Document{
		Popularity: *popularity,
		Title:      *ptitle,
		Body:       in,
		ObjectID:   id,
		Tags:       *tags,
		Indexed:    0,
	}

	store.BulkUpsert([]*data.Document{doc})
}
