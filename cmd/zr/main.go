package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jackdoe/go-pager"
	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage:\n\nzr [-top 10] [-k so,man,su] query string\n\n")
	flag.PrintDefaults()
	os.Exit(2)
}

type scored struct {
	rowID      int32
	score      float32
	popularity int32
}

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "root")
	kind := flag.String("k", "godoc,man,su,so", "csv list of indexes to search")
	topN := flag.Int("top", 1, "show top N question threads")
	debug := flag.Bool("debug", false, "show debug info")
	flag.Usage = usage
	flag.Parse()

	query := strings.Join(flag.Args(), " ")
	if query == "" {
		usage()
	}

	less, close := pager.Pager("less", "more")
	defer close()

	for i, v := range strings.Split(*kind, ",") {
		if v == "" {
			continue
		}

		store := data.NewStore(*root, v)

		hits := []scored{}
		limit := *topN

		q := store.MakeQuery("body", query)
		if *debug {
			fmt.Fprintf(less, "query: <%s> %v\n", query, q.String())
		}
		store.Dir.Foreach(q, func(did int32, score float32) {
			var h scored
			popularity := store.ReadWeight(did)

			h.rowID = did
			h.popularity = popularity
			h.score = score

			hits = append(hits, h)
		})

		sort.Slice(hits, func(i, j int) bool {
			sa := int(hits[i].score + 0.5)
			sb := int(hits[j].score + 0.5)

			pa := hits[i].popularity
			pb := hits[j].popularity

			if sa == sb {
				return pb < pa
			}
			return sb < sa
		})

		if len(hits) > limit {
			hits = hits[:limit]
		}

		if i >= 1 {
			fmt.Fprintf(less, "\n%s\n\n", util.Center(v, '█'))
		}
		for _, h := range hits {
			var doc data.Document
			if err := store.DB.Model(data.Document{}).Find(&doc, h.rowID).Error; err != nil {
				panic(err)
			}
			if *debug {
				fmt.Fprintf(less, "HIT: %+v\n", h)
			}
			_, _ = less.Write(util.Decompress(doc.Body))
		}
	}
}
