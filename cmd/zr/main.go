package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	iq "github.com/rekki/go-query"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage:\n\nzr [-top 10] [-root index root] [-only-title] [-only-body] [-tags go,c,..] query string\n\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "index root")
	onlyTitle := flag.Bool("only-title", false, "only search in the post's title (only questions have titles)")
	onlyBody := flag.Bool("only-body", false, "only search in the post's body")
	onlyQuestions := flag.Bool("only-questions", false, "only search in the questions")
	onlyAnswers := flag.Bool("only-answers", false, "only search in the answers")
	onlyAccepted := flag.Bool("only-accepted", false, "only questions and their accepted answer")
	tags := flag.String("tags", "", "search only in those tags e.g. c,go,php")
	topN := flag.Int("top", 1, "show top N question threads")
	debug := flag.Bool("debug", false, "debug")
	flag.Usage = usage
	flag.Parse()

	if *root == "" {
		log.Fatal("need root")
	}

	t0 := time.Now()

	store, err := data.NewStore(*root, 0)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	query := strings.Join(flag.Args(), " ")
	if query == "" {
		usage()
	}

	queries := []iq.Query{}

	if *onlyTitle {
		queries = append(queries, iq.And(store.Dir.Terms("title", query)...).SetBoost(2))
	} else if *onlyBody {
		queries = append(queries, iq.And(store.Dir.Terms("body", query)...).SetBoost(2))
	} else {
		queries = append(queries, iq.DisMax(
			0.01,
			iq.And(store.Dir.Terms("title", query)...).SetBoost(2),
			iq.And(store.Dir.Terms("body", query)...).SetBoost(1),
		))
	}

	if *onlyAccepted {
		queries = append(queries, iq.And(store.Dir.Terms("accepted", "true")...))
	}

	if *onlyQuestions {
		queries = append(queries, iq.And(store.Dir.Terms("type", "question")...))
	}

	if *onlyAnswers {
		queries = append(queries, iq.And(store.Dir.Terms("type", "answer")...))
	}

	if *tags != "" {
		for _, v := range strings.Split(*tags, ",") {
			if len(v) > 0 {
				queries = append(queries, iq.And(store.Dir.Terms("tags", v)...).SetBoost(1))
			}
		}
	}

	var q iq.Query
	if len(queries) == 1 {
		q = queries[1]
	} else {
		q = iq.And(queries...)
	}
	if *debug {
		log.Printf("QUERY: %s", q.String())
	}
	type hit struct {
		score float32
		id    int32
	}

	scored := []hit{}
	limit := *topN
	total := 0

	store.Dir.Foreach(q, func(did int32, score float32) {
		var h hit
		soscore, acceptedAnswerID, viewCount := store.ReadWeight(did)

		h.id = did

		h.score = float32(math.Log1p(float64(viewCount)))
		if acceptedAnswerID > 0 {
			h.score *= 10
		}

		if soscore < 0 {
			h.score *= float32(soscore)
		} else {
			h.score += float32(math.Log1p(float64(soscore)))
		}

		doInsert := false
		if len(scored) < limit {
			doInsert = true
		} else if scored[len(scored)-1].score < h.score {
			doInsert = true
		}

		if doInsert {
			if len(scored) < limit {
				scored = append(scored, h)
			}
			for i := 0; i < len(scored); i++ {
				if scored[i].score < h.score {
					copy(scored[i+1:], scored[i:])
					scored[i] = h
					break
				}
			}
		}
		total++
	})
	tookSoFar := time.Since(t0)

	seen := map[int32]bool{}
	for _, s := range scored {
		var p data.Post
		if err := store.DB.First(&p, "post_id = ?", s.id).Error; err != nil {
			panic(err)
		}

		var posts []data.Post
		var question data.Post
		if p.IsQuestion() {
			question = p
		} else {
			if err := store.DB.First(&question, "post_id = ?", p.ParentID).Error; err != nil {
				panic(err)
			}
		}

		if seen[question.PostID] {
			continue
		}

		fmt.Printf("%s", BannerLeft(30, "│", strings.Split(question.String(), "\n")))

		seen[question.PostID] = true

		if err := store.DB.Find(&posts, "parent_id = ?", p.PostID).Error; err != nil {
			panic(err)
		}

		sort.Slice(posts, func(i, j int) bool {
			a := posts[i]
			b := posts[j]

			scoreA := a.Score
			scoreB := b.Score

			if a.PostID == question.AcceptedAnswerID {
				scoreA += 10000
			}
			if b.PostID == question.AcceptedAnswerID {
				scoreB += 10000
			}

			return scoreB < scoreA
		})
		for _, post := range posts {
			if post.PostID == question.AcceptedAnswerID {
				post.Accepted = true
			}

			fmt.Printf("%s", BannerLeft(5, " ", strings.Split(post.String(), "\n")))
		}
	}
	fmt.Printf("\ntotal: %v, took: %v, without stdout: %v\n", total, time.Since(t0), tookSoFar)
}

func BannerLeft(topDashLen int, prefix string, s []string) string {
	out := "┌"
	for i := 0; i < topDashLen; i++ {
		out += "-"
	}
	out += "\n"

	for _, l := range s {
		out += prefix
		out += " "

		out += l
		out += "\n"
	}
	out += "└--"
	out += "\n"
	return out
}
