package util

import (
	"log"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

func Wait(client *meilisearch.Client, indexName string, id int64) {
	n := 0
	for {
		update, err := client.Updates(indexName).Get(id)
		if err != nil {
			panic(err)
		}
		if update.Status != meilisearch.UpdateStatusProcessed {
			if n > 0 {
				log.Printf(".. waiting for update %d to finish [ %v ], took so far: %v", id, update, time.Since(update.EnqueuedAt))
			}
			time.Sleep(1 * time.Second)
		}
		n++
		break
	}
}

func AddAndWait(client *meilisearch.Client, indexName string, docs interface{}) {
	r, err := client.Documents(indexName).AddOrReplace(docs)
	if err != nil {
		panic(err)
	}
	Wait(client, indexName, r.UpdateID)
}
