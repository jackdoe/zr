package data

import (
	"encoding/binary"
	"log"
	"os"
	"path"

	"github.com/jackdoe/zr/pkg/inv"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/rekki/go-query/util/analyzer"
	"github.com/rekki/go-query/util/index"
)

type Store struct {
	DB     *gorm.DB
	Dir    *inv.RocksIndex
	Weight *os.File
}

func NewStore(root string, maxfd int) (*Store, error) {
	err := os.MkdirAll(root, 0700)
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open("sqlite3", path.Join(root, "posts.db"))
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&Post{})

	weight, err := os.OpenFile(path.Join(root, "weight"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}
	os.MkdirAll(path.Join(root, "badger-inv"), 0700)
	di, err := inv.NewRocksIndex(path.Join(root, "rocks-inv"), map[string]*analyzer.Analyzer{
		"title": DefaultAnalyzer,
		"body":  DefaultAnalyzer,
		"tags":  index.IDAnalyzer,
	})
	if err != nil {
		panic(err)
	}

	return &Store{DB: db, Dir: di, Weight: weight}, nil
}

func (s *Store) Close() {
	s.DB.Close()
	s.Dir.Close()
	s.Weight.Close()
}

func (s *Store) WriteWeight(did int32, p Post) error {
	scoreB := make([]byte, 12)
	binary.LittleEndian.PutUint32(scoreB, uint32(p.Score))
	binary.LittleEndian.PutUint32(scoreB[4:], uint32(p.AcceptedAnswerID))
	binary.LittleEndian.PutUint32(scoreB[8:], uint32(p.ViewCount))

	_, err := s.Weight.WriteAt(scoreB, int64(p.PostID)*int64(len(scoreB)))
	return err
}

func (s *Store) ReadWeight(did int32) (int, int32, int) {
	b := make([]byte, 12)
	_, err := s.Weight.ReadAt(b, int64(did)*int64(len(b)))
	if err != nil {
		return 0, 0, 0
	}

	soscore := int(binary.LittleEndian.Uint32(b))
	acceptedAnswerID := int32(binary.LittleEndian.Uint32(b[4:]))
	viewCount := int(binary.LittleEndian.Uint32(b[8:]))

	return soscore, acceptedAnswerID, viewCount
}
