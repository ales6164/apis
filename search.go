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

func ClearIndex(ctx context.Context, indexName string) error {
	index, err := search.Open(indexName)
	if err != nil {
		return err
	}

	var ids []string
	for t := index.List(ctx, &search.ListOptions{IDsOnly: true}); ; {
		var doc interface{}
		id, err := t.Next(&doc)
		if err == search.Done {
			break
		}
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	return index.DeleteMulti(ctx, ids)
}
