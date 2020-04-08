package data

import (
	"encoding/binary"
	"log"
	"os"
	"path"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/rekki/go-query/util/analyzer"
	"github.com/rekki/go-query/util/index"
)

type Store struct {
	DB     *gorm.DB
	Dir    *index.DirIndex
	Weight *os.File
}

func NewStore(root string, kind string, maxfd int) (*Store, error) {
	err := os.MkdirAll(root, 0700)
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open("sqlite3", path.Join(root, kind, "main.db"))
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&Document{})

	fdc := index.NewFDCache(maxfd)

	weight, err := os.OpenFile(path.Join(root, kind, "weight"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}

	di := index.NewDirIndex(path.Join(root, kind, "inv"), fdc, map[string]*analyzer.Analyzer{
		"title": DefaultAnalyzer,
		"body":  DefaultAnalyzer,
		"tags":  index.IDAnalyzer,
	})

	di.DirHash = func(s string) string {
		return string(s[0]) + string(s[len(s)-1])
	}

	di.Lazy = false

	return &Store{DB: db, Dir: di, Weight: weight}, nil
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
