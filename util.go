package apis

import (
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"time"
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
)

const COST = 12

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(n int) string {
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

func ExpandMeta(ctx context.Context, output map[string]interface{}) (map[string]interface{}, map[string]interface{}) {
	var userMeta map[string]interface{}
	if meta, ok := output["meta"].(map[string]interface{}); ok {
		if key, ok := meta["createdBy"]; ok {
			//key, _ := datastore.DecodeKey(k.(string))

			if k, ok := key.(*datastore.Key); ok {
				user := new(User)
				if err := datastore.Get(ctx, k, user); err == nil {
					userMeta = user.Meta
					meta["createdBy"] = map[string]interface{}{
						"meta": user.Meta,
						"id":   k,
					}
				}
			}
		}
	}
	return output, userMeta
}

func decrypt(hash []byte, password []byte) error {
	defer clear(password)
	return bcrypt.CompareHashAndPassword(hash, password)
}

func crypt(password []byte) ([]byte, error) {
	defer clear(password)
	return bcrypt.GenerateFromPassword(password, COST)
}

func clear(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}
