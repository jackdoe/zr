package inv

import (
	"encoding/binary"
	"fmt"
	"strings"

	iq "github.com/rekki/go-query"
	"github.com/rekki/go-query/util/analyzer"
	"github.com/rekki/go-query/util/index"
	"github.com/tecbot/gorocksdb"
)

var ro = gorocksdb.NewDefaultReadOptions()
var wo = gorocksdb.NewDefaultWriteOptions()

type RocksIndex struct {
	perField          map[string]*analyzer.Analyzer
	root              string
	rocks             *gorocksdb.DB
	TotalNumberOfDocs int
}

type concatMergeOperator struct {
}

func (m *concatMergeOperator) Name() string { return "concat" }
func (m *concatMergeOperator) FullMerge(key, existingValue []byte, operands [][]byte) ([]byte, bool) {
	for _, o := range operands {
		existingValue = append(existingValue, o...)
	}
	return existingValue, true
}

func (m *concatMergeOperator) PartialMerge(key, leftOperand, rightOperand []byte) ([]byte, bool) {
	return append(leftOperand, rightOperand...), true
}

func NewRocksIndex(root string, perField map[string]*analyzer.Analyzer) (*RocksIndex, error) {
	if perField == nil {
		perField = map[string]*analyzer.Analyzer{}
	}

	filter := gorocksdb.NewBloomFilter(10)
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	bbto.SetFilterPolicy(filter)

	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)

	opts.SetMergeOperator(&concatMergeOperator{})

	db, err := gorocksdb.OpenDb(opts, root)
	if err != nil {
		return nil, err
	}
	return &RocksIndex{TotalNumberOfDocs: 1, rocks: db, root: root, perField: perField}, nil
}

func (d *RocksIndex) add(wb *gorocksdb.WriteBatch, k []byte, docs []int32) {
	b := make([]byte, 4*len(docs))
	for i, did := range docs {
		binary.LittleEndian.PutUint32(b[(i*4):], uint32(did))
	}
	wb.Merge(k, b)
}

func (d *RocksIndex) Index(docs ...index.DocumentWithID) error {
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

	wb := gorocksdb.NewWriteBatch()

	for f, docs := range todo {
		d.add(wb, []byte(f), docs)
	}

	return d.rocks.Write(wo, wb)

}

func (d *RocksIndex) Terms(field string, term string) []iq.Query {
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

func (d *RocksIndex) newTermQuery(field string, term string) iq.Query {
	if len(field) == 0 || len(term) == 0 {
		return iq.Term(d.TotalNumberOfDocs, fmt.Sprintf("broken(%s:%s)", field, term), []int32{})
	}

	data := []byte{}
	key := field + "/" + term
	value, err := d.rocks.Get(ro, []byte(key))

	if err == nil {
		data = value.Data()
		defer value.Free()
	}

	postings := make([]int32, len(data)/4)
	for i := 0; i < len(postings); i++ {
		from := i * 4
		postings[i] = int32(binary.LittleEndian.Uint32(data[from : from+4]))
	}
	return iq.Term(d.TotalNumberOfDocs, key, postings)
}

func (d *RocksIndex) Close() {
	d.rocks.Close()
}

func (d *RocksIndex) Foreach(query iq.Query, cb func(int32, float32)) {
	for query.Next() != iq.NO_MORE {
		did := query.GetDocId()
		score := query.Score()

		cb(did, score)
	}
}
