package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
	"github.com/rekki/go-query/util/index"
)

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "index root")
	atatime := flag.Int("at-a-time", 1000, "how many at a time")
	maxopen := flag.Int("max-fd", 1000, "max open fd")
	onlyAccepted := flag.Bool("only-accepted", false, "only questions with accepted answers")
	onlyWithAnswers := flag.Bool("only-with-answers", false, "only questions with at least 1 answer")
	onlyNScore := flag.Int("at-least-score", -1000, "only questions with at least that much score")
	onlyWithNViews := flag.Int("at-least-n-views", 0, "only question threads with at least N views")
	pprofBind := flag.String("pprof-bind", "", "bind pprof (e.g. localhost:6060)")
	flag.Parse()

	if *pprofBind != "" {
		go func() {
			log.Println(http.ListenAndServe(*pprofBind, nil))
		}()
	}

	if *root == "" {
		log.Fatal("need root")
	}

	store, err := data.NewStore(*root, *maxopen)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	n := 0
	t0 := time.Now()
	max := 43000000

	type Stats struct {
		NoAccept int
		NoAnswer int
		NoView   int
		NoScore  int
	}

	stats := Stats{}

	for {
		posts := []*data.Post{}
		if err := store.DB.Table("posts").Where("indexed = 0").Limit(*atatime).Order("post_id asc").Find(&posts).Error; err != nil {
			panic(err)
		}

		ids := []int32{}

		filtered := []*data.Post{}

		for _, p := range posts {
			ids = append(ids, p.PostID)

			acceptedAnswerID := p.AcceptedAnswerID
			viewCount := p.ViewCount
			if p.ParentID != 0 {
				var parent data.Post
				if err := store.DB.Find(&parent, p.ParentID).Error; err != nil {
					panic(err)
				}
				viewCount = parent.ViewCount
				acceptedAnswerID = parent.AcceptedAnswerID
			}

			noAccepted := acceptedAnswerID == 0
			noAnswers := p.PostTypeID == 1 && p.AnswerCount == 0

			if noAccepted {
				stats.NoAccept++
			}

			if noAnswers {
				stats.NoAnswer++
			}

			if viewCount < *onlyWithNViews {
				stats.NoView++
			}

			if p.Score < *onlyNScore {
				stats.NoScore++
			}

			if (noAccepted && *onlyAccepted) || (noAnswers && *onlyWithAnswers) {
				continue
			}

			if viewCount < *onlyWithNViews || p.Score < *onlyNScore {
				if p.IsQuestion() {
					continue
				}

				if acceptedAnswerID == p.PostID {
					// always index the accepted answer
					continue
				}
			}

			filtered = append(filtered, p)
		}
		if len(posts) == 0 {
			break
		}

		err := store.Dir.Index(toDocs(filtered)...)
		if err != nil {
			panic(err)
		}

		tx := store.DB.Begin()
		util.Chunked(100, len(ids), func(from, to int) {
			if err := tx.Table("posts").Where("post_id IN (?)", ids[from:to]).Updates(map[string]interface{}{"indexed": 1}).Error; err != nil {
				panic(err)
			}
		})
		if err := tx.Commit().Error; err != nil {
			panic(err)
		}

		n += len(ids)
		if n%1000 == 0 {
			took := time.Since(t0)
			perSecond := float64(n) / took.Seconds()
			eta := float64(max-n) / perSecond
			log.Printf("indexing ... %d [stats: %+v], per second: %.2f, ~ETA: %.2f hours (%d left)", n, stats, perSecond, eta/3600, max-n)
		}
	}
}

func toDocs(in []*data.Post) []index.DocumentWithID {
	out := make([]index.DocumentWithID, len(in))
	for i, v := range in {
		out[i] = index.DocumentWithID(v)
	}
	return out
}
