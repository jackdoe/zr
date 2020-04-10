package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	_ "net/http/pprof"
	"os"
	"path"
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

	store := data.NewStore(*root, *kind)
	defer store.Close()

	var doc data.Document
	if *rid != 0 {
		if err := store.DB.Model(data.Document{}).Where("row_id=?", *rid).First(&doc).Error; err != nil {
			panic(err)
		}
	} else {
		if err := store.DB.Model(data.Document{}).Where("object_id=?", *id).First(&doc).Error; err != nil {
			panic(err)
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

	dir := path.Join(*root, *kind, "inv", "body")

	sort.Strings(tokens)
	for idx, t := range tokens {
		p := path.Join(dir, store.Dir.DirHash(t), t)
		data, err := ioutil.ReadFile(p)
		if err != nil {
			panic(err)
		}
		postings := make([]int32, len(data)/4)
		for i := 0; i < len(postings); i++ {
			from := i * 4
			postings[i] = int32(binary.LittleEndian.Uint32(data[from : from+4]))
		}

		found := false
		for _, did := range postings {
			if did == doc.RowID {
				found = true
			}
		}
		fmt.Printf("%3d -> %s file: %s, postings: %v [%v]\n", idx, t, p, len(postings), found)
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
