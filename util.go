package apis

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/search"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
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

var queryFilters = regexp.MustCompile(`(?m)filters\[(?P<num>[^\]]+)\]\[(?P<nam>[^\]]+)\]`)

func getParams(url string) (paramsMap map[string]string) {
	match := queryFilters.FindStringSubmatch(url)
	paramsMap = make(map[string]string)
	for i, name := range queryFilters.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

// getHost tries its best to return the request host.
func getHost(r *http.Request) string {
	var host = r.URL.Host
	if r.URL.IsAbs() {
		host = r.Host
		// Slice off any port information.
		if i := strings.Index(host, ":"); i != -1 {
			host = host[:i]
		}
	}
	if len(host) == 0 {
		if appengine.IsDevAppServer() {
			host = appengine.DefaultVersionHostname(appengine.NewContext(r))
		}
	}
	return host
}

func getSchemeAndHost(r *http.Request) string {
	if r.Header.Get("X-AppEngine-Https") == "on" {
		return "https://" + getHost(r)
	}
	return "http://" + getHost(r)
}

func joinPath(p ...string) string {
	return "/" + strings.Join(p, "/")
}