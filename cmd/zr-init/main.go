package main

import (
	"flag"
	_ "net/http/pprof"
	"strings"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/meilisearch/meilisearch-go"
)

func main() {
	masterKey := flag.String("master-key", "zr", "master key")
	meiliURL := flag.String("meili", "http://127.0.0.1:7700", "meili search url")
	kind := flag.String("kind", "so,su,man", "csv list of indexes")
	flag.Parse()

	client := meilisearch.NewClient(meilisearch.Config{
		Host:   *meiliURL,
		APIKey: *masterKey,
	})

	for _, v := range strings.Split(*kind, ",") {
		if v == "" {
			continue
		}

		_, err := client.Indexes().Create(meilisearch.CreateIndexRequest{
			UID:        data.IndexName(v),
			PrimaryKey: "id",
		})
		if err != nil {
			panic(err)
		}

		_, err = client.Settings(data.IndexName(v)).UpdateRankingRules([]string{"attribute", "words", "proximity", "wordsPosition", "desc(popularity)"})
		if err != nil {
			panic(err)
		}
	}
}
