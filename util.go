package apis

import (
		"math/rand"
	"time"
	"github.com/ales6164/apis/kind"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/search"
)

const LetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const LetterNumberBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const NumberBytes = "0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(letterBytes string, n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}



// purges datastore and search of all entries
func ClearAllEntries(ctx context.Context, kind *kind.Kind) error {
	keys, _ := datastore.NewQuery(kind.Name).KeysOnly().GetAll(ctx, nil)
	if len(keys) > 0 {
		err := datastore.DeleteMulti(ctx, keys)
		if err != nil {
			return err
		}
	}
	return ClearIndex(ctx, kind.Name)
}

// clears search index
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
