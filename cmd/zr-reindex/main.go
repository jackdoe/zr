package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
)

func main() {
	batchSize := flag.Int("batch", 10000, "batch size")
	root := flag.String("root", util.GetDefaultRoot(), "root")
	kind := flag.String("k", "unknown", "kind of object (prependet to the id)")
	profBind := flag.String("pprof-bind", "", "bind pprof (e.g. localhost:6060)")
	flag.Parse()

	if *profBind != "" {
		go func() {
			log.Println(http.ListenAndServe(*profBind, nil))
		}()
	}

	if *root == "" {
		log.Fatal("root")
	}

	if *kind == "" {
		log.Fatal("kind")
	}

	for _, v := range strings.Split(*kind, ",") {
		if v == "" {
			continue
		}

		store := data.NewStore(*root, v)
		defer store.Close()
		store.Reindex(*batchSize)
	}
}
