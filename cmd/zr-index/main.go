package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
	"github.com/rekki/go-query/util/index"
)

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "index root")
	atatime := flag.Int("at-a-time", 1000, "how many at a time")
	nshards := flag.Int("shards", 2, "how many shards")
	maxopen := flag.Int("max-fd", 1000, "max open fd")
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

	store, err := data.NewStore(*root, *nshards, *maxopen)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	wg := sync.WaitGroup{}

	for shardID, shard := range store.Shards {
		wg.Add(1)
		go func(shardID int, shard *index.DirIndex) {
			work(*atatime, store, shardID, shard)
			wg.Done()
		}(shardID, shard)
	}
	wg.Wait()
}

func work(atatime int, store *data.Store, shardID int, shard *index.DirIndex) {
	n := 0
	t0 := time.Now()
	max := 43000000 / len(store.Shards)

	for {
		posts := []*data.Post{}
		store.RLock()
		if err := store.DB.Table("posts").Where("indexed = 0 and post_id % ? = ?", len(store.Shards), shardID).Limit(atatime).Order("post_id asc").Find(&posts).Error; err != nil {
			panic(err)
		}
		store.RUnlock()

		ids := []int32{}

		for _, p := range posts {
			ids = append(ids, p.PostID)
		}
		if len(posts) == 0 {
			break
		}

		err := shard.Index(toDocs(posts)...)
		if err != nil {
			panic(err)
		}

		store.Lock()
		tx := store.DB.Begin()
		util.Chunked(100, len(ids), func(from, to int) {
			if err := tx.Table("posts").Where("post_id IN (?)", ids[from:to]).Updates(map[string]interface{}{"indexed": 1}).Error; err != nil {
				panic(err)
			}
		})

		if err := tx.Commit().Error; err != nil {
			panic(err)
		}
		store.Unlock()

		n += len(posts)
		if n%1000 == 0 {
			took := time.Since(t0)
			perSecond := float64(n) / took.Seconds()
			eta := float64(max-n) / perSecond
			log.Printf("[%d] indexing ... %d, per second: %.2f, ~ETA: %.2f hours (%d left)", shardID, n, perSecond, eta/3600, max-n)
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
