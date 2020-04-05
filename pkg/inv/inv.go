package inv

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger"
	iq "github.com/rekki/go-query"
	"github.com/rekki/go-query/util/analyzer"
	"github.com/rekki/go-query/util/index"
)

type BadgerIndex struct {
	perField          map[string]*analyzer.Analyzer
	root              string
	bad               *badger.DB
	TotalNumberOfDocs int
}

func NewBadgerIndex(root string, perField map[string]*analyzer.Analyzer) (*BadgerIndex, error) {
	if perField == nil {
		perField = map[string]*analyzer.Analyzer{}
	}

	b, err := badger.Open(badger.DefaultOptions(root))
	if err != nil {
		return nil, err
	}

	return &BadgerIndex{TotalNumberOfDocs: 1, bad: b, root: root, perField: perField}, nil
}

func (d *BadgerIndex) add(tx *badger.Txn, k []byte, docs []int32) error {

	item, _ := tx.Get([]byte(k))
	if item != nil {
		return item.Value(func(val []byte) error {
			b := make([]byte, 4*len(docs)+len(val))
			copy(b, val)
			for i, did := range docs {
				binary.LittleEndian.PutUint32(b[(i*4)+len(val):], uint32(did))
			}
			entry := badger.NewEntry(k, b)
			return tx.SetEntry(entry)
		})
	}

	b := make([]byte, 4*len(docs))
	for i, did := range docs {
		binary.LittleEndian.PutUint32(b[(i*4):], uint32(did))
	}
	entry := badger.NewEntry(k, b)
	return tx.SetEntry(entry)
}

func (d *BadgerIndex) Index(docs ...index.DocumentWithID) error {
	var sb strings.Builder

	todo := map[string][]int32{}
	for _, doc := range docs {
		did := doc.DocumentID()

		fields := doc.IndexableFields()
		for field, value := range fields {
			if len(field) == 0 {
				continue
			}

			analyzer, ok := d.perField[field]
			if !ok {
				analyzer = index.DefaultAnalyzer
			}
			for _, v := range value {
				tokens := analyzer.AnalyzeIndex(v)
				for _, t := range tokens {
					if len(t) == 0 {
						continue
					}

					sb.WriteString(field)
					sb.WriteRune('/')
					sb.WriteString(t)
					s := sb.String()
					todo[s] = append(todo[s], did)
					sb.Reset()
				}
			}
		}
	}

	return d.bad.Update(func(txn *badger.Txn) error {
		for f, docs := range todo {
			err := d.add(txn, []byte(f), docs)
			if err != nil {
				return err
			}

		}
		return nil
	})

}

func (d *BadgerIndex) Terms(field string, term string) []iq.Query {
	analyzer, ok := d.perField[field]
	if !ok {
		analyzer = index.DefaultAnalyzer
	}
	tokens := analyzer.AnalyzeSearch(term)
	queries := []iq.Query{}
	for _, t := range tokens {
		queries = append(queries, d.newTermQuery(field, t))
	}
	return queries
}

func (d *BadgerIndex) newTermQuery(field string, term string) iq.Query {
	if len(field) == 0 || len(term) == 0 {
		return iq.Term(d.TotalNumberOfDocs, fmt.Sprintf("broken(%s:%s)", field, term), []int32{})
	}

	data := []byte{}

	key := field + "/" + term
	_ = d.bad.View(func(tx *badger.Txn) error {
		item, _ := tx.Get([]byte(key))
		_ = item.Value(func(val []byte) error {
			data = append(data, val...)
			return nil
		})
		return nil
	})

	postings := make([]int32, len(data)/4)
	for i := 0; i < len(postings); i++ {
		from := i * 4
		postings[i] = int32(binary.LittleEndian.Uint32(data[from : from+4]))
	}
	return iq.Term(d.TotalNumberOfDocs, key, postings)
}

func (d *BadgerIndex) Close() {
	d.bad.Close()
}

func (d *BadgerIndex) Foreach(query iq.Query, cb func(int32, float32)) {
	for query.Next() != iq.NO_MORE {
		did := query.GetDocId()
		score := query.Score()

		cb(did, score)
	}
}
