package data

import (
	"runtime"
	"sync"

	"github.com/dgryski/go-metro"
)

type Store struct {
	Shards []*Shard
}

// having 16 shards for stackoverflow means 2.5 million documents per shard
var N_SHARDS = uint32(16)

func NewStore(root string, kind string) *Store {
	shards := []*Shard{}

	for i := uint32(0); i < N_SHARDS; i++ {
		shard := NewShard(root, kind, i)
		shards = append(shards, shard)
	}

	return &Store{Shards: shards}
}

func (s *Store) Parallel(cb func(int, *Shard)) {
	wg := sync.WaitGroup{}

	max := runtime.GOMAXPROCS(0)
	wait := make(chan bool, max)
	for i := 0; i < max; i++ {
		wait <- true
	}

	for shardID, shard := range s.Shards {
		<-wait
		wg.Add(1)
		go func(shardID int, shard *Shard) {
			cb(shardID, shard)
			wait <- true
			wg.Done()
		}(shardID, shard)
	}

	wg.Wait()
	close(wait)
}

func (s *Store) Reindex(batchSize int) {
	s.Parallel(func(_id int, shard *Shard) {
		shard.Reindex(batchSize)
	})
}

func (s *Store) BulkUpsert(batch []*Document) {
	perShard := map[uint32][]*Document{}

	for _, d := range batch {
		h := metro.Hash64Str(d.ObjectID, 0)
		shard := uint32(h) % N_SHARDS
		perShard[shard] = append(perShard[shard], d)
	}

	s.Parallel(func(shardID int, shard *Shard) {
		shard.BulkUpsert(perShard[uint32(shardID)])
	})
}

func (s *Store) ShardFor(objectID string) *Shard {
	return s.Shards[int(s.ShardID(objectID))]
}

func (s *Store) ShardID(objectID string) uint32 {
	h := metro.Hash64Str(objectID, 0)
	shard := uint32(h) % N_SHARDS
	return uint32(shard)
}

func (s *Store) Close() {
	for _, sh := range s.Shards {
		sh.DB.Close()
		sh.Dir.Close()
		sh.Weight.Close()
	}
}
