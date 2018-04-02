package apis

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/search"
)

func PutToIndex(ctx context.Context, indexName string, documentId string, value interface{}) error {
	index, err := search.Open(indexName)
	if err != nil {
		return err
	}
	_, err = index.Put(ctx, documentId, value)
	return err
}

func GetFromIndex(ctx context.Context, indexName string, documentId string, dst interface{}) error {
	index, err := search.Open(indexName)
	if err != nil {
		return err
	}
	return index.Get(ctx, documentId, dst)
}

func OpenIndex(name string) (*search.Index, error) {
	return search.Open(name)
}
