package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

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
	doc        *data.Document
}
type ByScore []scored

func (hits ByScore) Len() int      { return len(hits) }
func (hits ByScore) Swap(i, j int) { hits[i], hits[j] = hits[j], hits[i] }
func (hits ByScore) Less(i, j int) bool {

	sa := int(hits[i].score + 0.5)
	sb := int(hits[j].score + 0.5)

	pa := hits[i].popularity
	pb := hits[j].popularity

	if sa == sb {
		return pb < pa
	}
	return sb < sa
}

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "root")
	kind := flag.String("k", "man,su,so,godoc", "csv list of indexes to search")
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
	limit := *topN

	for _, v := range strings.Split(*kind, ",") {
		if v == "" {
			continue
		}

		store := data.NewStore(*root, v)
		total := []scored{}
		lock := sync.Mutex{}

		// get the topN of each shard
		store.Parallel(func(shardID int, shard *data.Shard) {
			hits := []scored{}

			q := shard.MakeQuery("body", query)
			if *debug {
				fmt.Fprintf(less, "shard %d query: <%s> %v\n", shardID, query, q.String())
			}
			shard.Dir.Foreach(q, func(did int32, score float32) {
				var h scored
				popularity := shard.ReadWeight(did)

				h.rowID = did
				h.popularity = popularity
				h.score = score

				hits = append(hits, h)
			})

			sort.Sort(ByScore(hits))

			if len(hits) > limit {
				hits = hits[:limit]
			}

			for i := range hits {
				var doc data.Document
				if err := shard.DB.Model(data.Document{}).Find(&doc, hits[i].rowID).Error; err != nil {
					panic(err)
				}
				hits[i].doc = &doc
			}

			lock.Lock()
			total = append(total, hits...)
			lock.Unlock()
		})

		// get topN of sharded topN
		sort.Sort(ByScore(total))
		if len(total) > limit {
			total = total[:limit]
		}

		if len(total) > 0 {
			fmt.Fprintf(less, "\n%s\n\n", util.Center(v, '█'))
		}
		for _, h := range total {
			if *debug {
				fmt.Fprintf(less, "HIT: %+v\n", h)
			}
			_, _ = less.Write(util.Decompress(h.doc.Body))
		}
	}
}
