package apis

import (
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


func ContainsScope(arr []string, scopes ...string) bool {
	for _, scp := range scopes {
		for _, r := range arr {
			if r == scp {
				return true
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
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
	var host = r.Host
	if len(host) == 0 {

	} else {
		if i := strings.Index(host, ":"); i != -1 {
			host = host[:i]
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
	return strings.Join(p, "/")
}
