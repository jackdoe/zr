package data

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jackdoe/zr/pkg/util"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	iq "github.com/rekki/go-query"
	"github.com/rekki/go-query/util/analyzer"
	"github.com/rekki/go-query/util/index"
)

type Store struct {
	DB     *gorm.DB
	Dir    *index.DirIndex
	Weight *os.File
	root   string
	kind   string
}

type FDCache struct {
	created map[string]bool
}

func (x *FDCache) Close() {
}

func (x *FDCache) Use(fn string, createFile func(fn string) (*os.File, error), cb func(*os.File) error) error {
	dir := path.Dir(fn)
	if !x.created[dir] {
		_ = os.MkdirAll(dir, 0700)
		x.created[dir] = true
	}

	f, err := createFile(fn)

	if err != nil {
		return err
	}

	defer f.Close()
	return cb(f)
}

func NewStore(root string, kind string) *Store {
	err := os.MkdirAll(path.Join(root, kind), 0700)
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open("sqlite3", path.Join(root, kind, "main.db"))
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&Document{})

	fdc := &FDCache{created: map[string]bool{}}

	weight, err := os.OpenFile(path.Join(root, kind, "weight"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}

	invRoot := path.Join(root, kind, "inv")
	di := index.NewDirIndex(invRoot, fdc, map[string]*analyzer.Analyzer{
		"body": DefaultAnalyzer,
	})

	di.DirHash = func(s string) string {
		return string([]byte{s[0], s[len(s)-1]})
	}

	di.Lazy = false

	return &Store{DB: db, Dir: di, Weight: weight, kind: kind, root: root}
}

func andOrFirst(q []iq.Query) iq.Query {
	if len(q) == 1 {
		return q[0]
	}

	return iq.And(q...)
}

func (s *Store) MakeQuery(field string, query string) iq.Query {
	or := []iq.Query{}

	normalizedQuery := ascii(strings.TrimSpace(query))
	ws := strings.Split(normalizedQuery, " ")

	for i := 0; i < MAX_CHUNKS; i++ {
		and := []iq.Query{}
		for _, w := range ws {
			term := trim(fmt.Sprintf("%s_%d", w, i))
			q := s.Dir.NewTermQuery(field, term)
			and = append(and, q)
		}
		or = append(or, iq.Constant(float32(1+MAX_CHUNKS-i), andOrFirst(and)))
	}

	return iq.DisMax(0.01, or...)
}

func toDocumentWithID(in []*Document) []index.DocumentWithID {
	out := make([]index.DocumentWithID, 0, len(in))
	for _, v := range in {
		out = append(out, index.DocumentWithID(v))
	}
	return out
}

func (s *Store) BulkUpsert(batch []*Document) {
	if len(batch) == 0 {
		return
	}

	tx := s.DB.Begin()

	for _, d := range batch {
		s.Upsert(tx, d)
	}

	if err := tx.Commit().Error; err != nil {
		panic(err)
	}
}

func (s *Store) Reindex(batchSize int) {
	invp := path.Join(s.root, s.kind, "inv")
	log.Printf("removing %v", invp)
	_ = os.RemoveAll(invp)
	_ = os.MkdirAll(invp, 0700)

	log.Printf("setting all documents as not indexed")
	if err := s.DB.Table("documents").Where("indexed = 1").Updates(map[string]interface{}{"indexed": 0}).Error; err != nil {
		panic(err)
	}

	cnt := 0
	if err := s.DB.Table("documents").Count(&cnt).Error; err != nil {
		panic(err)
	}

	idx := 0
	processed := 0
	t0 := time.Now()
	for {
		docs := []*Document{}
		if err := s.DB.Table("documents").Where("indexed = 0").Limit(batchSize).Order("row_id asc").Find(&docs).Error; err != nil {
			panic(err)
		}

		if len(docs) == 0 {
			return
		}

		ids := make([]int32, 0, len(docs))

		for _, d := range docs {
			ids = append(ids, d.RowID)
		}

		err := s.Dir.Index(toDocumentWithID(docs)...)
		if err != nil {
			panic(err)
		}

		tx := s.DB.Begin()
		util.Chunked(100, len(ids), func(from, to int) {
			if err := tx.Table("documents").Where("row_id IN (?)", ids[from:to]).Updates(map[string]interface{}{"indexed": 1}).Error; err != nil {
				panic(err)
			}
		})
		if err := tx.Commit().Error; err != nil {
			panic(err)
		}
		idx++
		processed += len(ids)
		took := time.Since(t0)
		perSecond := float64(len(ids)) / took.Seconds()
		log.Printf("...left: %d, per second: %.2f, took: %v, processed: %d", cnt-processed, perSecond, took, processed)
		t0 = time.Now()
	}
}

func (s *Store) Upsert(tx *gorm.DB, d *Document) *Document {
	rid := Document{}

	tx.Model(Document{}).Select("row_id").Where("object_id = ?", d.ObjectID).First(&rid)

	if rid.RowID > 0 {
		d.RowID = rid.RowID
	} else {
		d.RowID = 0
	}

	if err := tx.Model(Document{}).Save(d).Error; err != nil {
		panic(err)
	}

	err := s.WriteWeight(d.RowID, int32(d.Popularity))

	if err != nil {
		panic(err)
	}

	return d
}

func (s *Store) Close() {
	s.DB.Close()
	s.Dir.Close()
	s.Weight.Close()
}

func (s *Store) WriteWeight(did int32, w int32) error {
	b := []byte{0, 0, 0, 0}
	binary.LittleEndian.PutUint32(b, uint32(w))

	_, err := s.Weight.WriteAt(b, int64(did)*int64(len(b)))
	return err
}

func (s *Store) ReadWeight(did int32) int32 {
	b := []byte{0, 0, 0, 0}
	_, err := s.Weight.ReadAt(b, int64(did)*int64(len(b)))
	if err != nil {
		return 0
	}

	return int32(binary.LittleEndian.Uint32(b))
}
