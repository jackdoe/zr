package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
)

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "index root")
	posts := flag.String("posts", "", "path to Posts.xml")
	profBind := flag.String("prof-bind", "", "bind pprof (e.g. localhost:6060)")
	flag.Parse()
	if *profBind != "" {
		go func() {
			log.Println(http.ListenAndServe(*profBind, nil))
		}()
	}

	if *root == "" {
		log.Fatal("need root")
	}

	if *posts == "" {
		log.Fatal("need posts")
	}

	store, err := data.NewStore(*root, 0)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	n := 0
	t0 := time.Now()
	max := 43000000
	tx := store.DB.Begin()
	skip := 0
	err = data.DecodeFile(*posts, func(p data.Post) error {
		cnt := 0
		tx.Table("posts").Where("post_id = ?", p.PostID).Count(&cnt)
		if cnt != 0 {
			skip++

			if skip%10000 == 0 {
				took := time.Since(t0)
				perSecond := float64(skip) / took.Seconds()
				eta := float64(max-skip) / perSecond

				log.Printf("skipping ...  %d, per second: %.2f, ~ETA: %.2f hours (%d left)", skip, perSecond, eta/3600, max-skip)

			}
			return nil
		}

		n++
		if err := tx.FirstOrCreate(p).Error; err != nil {
			panic(err.Error)
		}

		if p.ParentID != 0 {
			_, _, viewCount := store.ReadWeight(p.ParentID)
			p.ViewCount = viewCount / 10
		}
		if err := store.WriteWeight(p.PostID, p); err != nil {
			panic(err)
		}

		if n%1000 == 0 {
			if err := tx.Commit().Error; err != nil {
				panic(err)
			}
			took := time.Since(t0)
			perSecond := float64(n) / took.Seconds()
			eta := float64(max-n) / perSecond
			log.Printf("storing ... %d, per second: %.2f, ~ETA: %.2f hours (%d left)", n, perSecond, eta/3600, max-n)
			tx = store.DB.Begin()
			t0 = time.Now()
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
	if err := tx.Commit().Error; err != nil {
		panic(err)
	}
}
