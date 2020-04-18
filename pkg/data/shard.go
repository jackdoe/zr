package data

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	inv "github.com/jackdoe/go-query-sql"
	"github.com/jackdoe/zr/pkg/util"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	iq "github.com/rekki/go-query"
	analyzer "github.com/rekki/go-query-analyze"

	index "github.com/rekki/go-query-index"
)

type Shard struct {
	DB     *gorm.DB
	Dir    *inv.LiteIndex
	Weight *os.File
	ID     uint32
}

func NewShard(root string, kind string, id uint32) *Shard {
	sid := fmt.Sprintf("shard_%d", id)
	err := os.MkdirAll(path.Join(root, kind, sid), 0700)
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open("sqlite3", path.Join(root, kind, sid, "main.db"))
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&Document{})

	weight, err := os.OpenFile(path.Join(root, kind, sid, "weight"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatal(err)
	}

	invDB, err := sql.Open("sqlite3", path.Join(root, kind, sid, "inv.db"))
	if err != nil {
		log.Fatal(err)
	}

	idx, err := inv.NewLiteIndex(invDB, inv.SQLITE3, "inv", map[string]*analyzer.Analyzer{
		"body": DefaultAnalyzer,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &Shard{DB: db, Dir: idx, Weight: weight, ID: id}
}

func andOrFirst(q []iq.Query) iq.Query {
	if len(q) == 1 {
		return q[0]
	}

	return iq.And(q...)
}

func OrOrFirst(q []iq.Query) iq.Query {
	if len(q) == 1 {
		return q[0]
	}

	return iq.Or(q...)
}

func DisOrFirst(tie float32, q []iq.Query) iq.Query {
	if len(q) == 1 {
		return q[0]
	}

	return iq.DisMax(tie, q...)
}

func (s *Shard) MakeQuery(field string, query string) iq.Query {
	or := []iq.Query{}

	splitted := strings.Split(query, " ")
	good := []string{}
	bad := []string{}
	for _, v := range splitted {
		if strings.HasPrefix(v, "-") {
			bad = append(bad, strings.TrimPrefix(v, "-"))
		} else {
			good = append(good, v)
		}
	}

	normalizedQuery := ascii(strings.TrimSpace(strings.Join(good, " ")))
	ws := strings.Split(normalizedQuery, " ")

	for i := 0; i < MAX_CHUNKS; i++ {
		and := []iq.Query{}
		for _, w := range ws {
			if len(w) > 0 {
				if len(w) > MAX_TOKEN_SIZE {
					w = w[:MAX_TOKEN_SIZE]
				}
				term := fmt.Sprintf("%s_%d", w, i)
				q := s.Dir.NewTermQuery(field, term)
				and = append(and, q)
			}
		}
		or = append(or, iq.Constant(float32(1+MAX_CHUNKS-i), andOrFirst(and)))
	}

	normalizedQuery = ascii(strings.TrimSpace(strings.Join(bad, " ")))
	ws = strings.Split(normalizedQuery, " ")
	not := []iq.Query{}
	for i := 0; i < MAX_CHUNKS; i++ {
		for _, w := range ws {
			if len(w) > 0 {
				term := fmt.Sprintf("%s_%d", w, i)
				q := s.Dir.NewTermQuery(field, term)
				not = append(not, q)
			}
		}
	}

	dis := DisOrFirst(0.01, or)
	if len(not) > 0 {
		return iq.AndNot(OrOrFirst(not), dis)
	}

	return dis
}

func toDocumentWithID(in []*Document) []index.DocumentWithID {
	out := make([]index.DocumentWithID, 0, len(in))
	for _, v := range in {
		out = append(out, index.DocumentWithID(v))
	}
	return out
}

func (s *Shard) BulkUpsert(batch []*Document) {
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

func (s *Shard) Reindex(batchSize int) {
	log.Printf("[shard %d] truncating inverted", s.ID)
	if err := s.Dir.Truncate(); err != nil {
		panic(err)
	}

	log.Printf("[shard %d] setting all documents as not indexed", s.ID)
	if err := s.DB.Table("documents").Where("indexed = 1").Updates(map[string]interface{}{"indexed": 0}).Error; err != nil {
		panic(err)
	}

	cnt := 0
	if err := s.DB.Table("documents").Count(&cnt).Error; err != nil {
		panic(err)
	}

	log.Printf("[shard %d] starting reindex for %d documents", s.ID, cnt)
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
			d.Body = util.Decompress(d.Body)
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
		log.Printf("[shard: %d] ...left: %d, per second: %.2f, took: %v, processed: %d", s.ID, cnt-processed, perSecond, took, processed)
		t0 = time.Now()
	}
}

func (s *Shard) Upsert(tx *gorm.DB, in *Document) *Document {
	rid := Document{}

	tx.Model(Document{}).Select("row_id").Where("object_id = ?", in.ObjectID).First(&rid)

	d := *in

	if rid.RowID > 0 {
		d.RowID = rid.RowID
	} else {
		d.RowID = 0
	}

	d.Body = util.Compress(d.Body)
	if err := tx.Model(Document{}).Save(&d).Error; err != nil {
		panic(err)
	}

	if d.Popularity > 0 {
		err := s.WriteWeight(d.RowID, int32(d.Popularity))

		if err != nil {
			panic(err)
		}
	}

	return &d
}

func (s *Shard) Close() {
	s.DB.Close()
	s.Dir.Close()
	s.Weight.Close()
}

func (s *Shard) WriteWeight(did int32, w int32) error {
	b := []byte{0, 0, 0, 0}
	binary.LittleEndian.PutUint32(b, uint32(w))

	_, err := s.Weight.WriteAt(b, int64(did)*int64(len(b)))
	return err
}

func (s *Shard) ReadWeight(did int32) int32 {
	b := []byte{0, 0, 0, 0}
	_, err := s.Weight.ReadAt(b, int64(did)*int64(len(b)))
	if err != nil {
		return 0
	}

	return int32(binary.LittleEndian.Uint32(b))
}
