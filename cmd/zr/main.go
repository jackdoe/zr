package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackdoe/zr/pkg/data"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/meilisearch/meilisearch-go"
	iq "github.com/rekki/go-query"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage:\n\nzr [-top 10] [-kind so,man,su] query string\n\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func andOrFirst(q []iq.Query) iq.Query {
	if len(q) == 1 {
		return q[0]
	}

	return iq.And(q...)
}
func main() {
	masterKey := flag.String("master-key", "zr", "master key")
	meiliURL := flag.String("meili", "http://127.0.0.1:7700", "meili search url")
	kind := flag.String("kind", "so,su,man", "csv list of indexes to search")
	topN := flag.Int("top", 1, "show top N question threads")
	flag.Usage = usage
	flag.Parse()

	var client = meilisearch.NewClient(meilisearch.Config{
		Host:   *meiliURL,
		APIKey: *masterKey,
	})

	query := strings.Join(flag.Args(), " ")
	if query == "" {
		usage()
	}

	for _, v := range strings.Split(*kind, ",") {
		if v == "" {
			continue
		}

		resp, err := client.Search(data.IndexName(v)).Search(meilisearch.SearchRequest{
			Query: query,
			Limit: int64(*topN),
		})

		if err != nil {
			panic(err)
		}

		for _, h := range resp.Hits {
			var d data.Document
			b, err := json.Marshal(h)
			if err != nil {
				panic(err)
			}

			err = json.Unmarshal(b, &d)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s", d.Body)
		}
	}
}
