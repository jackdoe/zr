package main

import (
	"flag"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"sort"
	"strings"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
)

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "root")
	kind := flag.String("k", "unknown", "kind of object (prependet to the id)")
	id := flag.String("id", "", "object_id")
	rid := flag.Int("rid", 0, "row id")
	dumpPostings := flag.Bool("dump-postings", false, "dump postings")
	flag.Parse()

	if *root == "" {
		log.Fatal("root")
	}

	if *id == "" && *rid == 0 {
		log.Fatal("id")
	}

	if *kind == "" {
		log.Fatal("kind")
	}

	sharded := data.NewStore(*root, *kind)
	defer sharded.Close()

	for _, store := range sharded.Shards {
		var doc data.Document
		if *rid != 0 {
			if err := store.DB.Model(data.Document{}).Where("row_id=?", *rid).First(&doc).Error; err != nil {
				continue
			}
		} else {
			if err := store.DB.Model(data.Document{}).Where("object_id=?", *id).First(&doc).Error; err != nil {
				continue
			}
		}

		doc.Body = util.Decompress(doc.Body)

		fmt.Printf(" TITLE:      %s\n", doc.Title)
		fmt.Printf(" TAGS:       %s\n", doc.Tags)
		fmt.Printf(" DOC ID:     %d\n", doc.RowID)
		fmt.Printf(" OBJECT ID:  %s\n", doc.ObjectID)
		fmt.Printf(" INDEXED:    %d\n", doc.Indexed)
		fmt.Printf("%s\n\n", strings.Repeat("*", 80))
		os.Stdout.Write(doc.Body)

		tokens := data.DefaultAnalyzer.AnalyzeIndex(string(doc.Body))

		sort.Strings(tokens)
		for idx, t := range tokens {
			postings := store.Dir.Postings("body/" + t)
			found := false
			for _, did := range postings {
				if did == doc.RowID {
					found = true
				}
			}
			fmt.Printf("%3d -> %s postings: %v [%v]\n", idx, t, len(postings), found)
			if *dumpPostings {
				for i, did := range postings {
					if did == doc.RowID {
						fmt.Printf("\t%10d: >>%d<<\n", i, did)
					} else {
						fmt.Printf("\t%10d: %d\n", i, did)
					}
				}

			}
		}
	}
}
