package utils

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	savedCachePath = "/tmp/go-tmp/"
)

func init() {
	os.Mkdir(savedCachePath, 0750)
}

type CacheResponse struct {
	E_at       time.Time
	Cache_path string
	*http.Response
}

/**
 * NewCacheResponse
 *
 * Creates a new cache response by fetching the resUrl parameter value.
 */
func NewCacheResponse(resUrl string) (cacheRes *CacheResponse, key string) {
	key = getSha512Sum(resUrl)

	if cacheRes = getCache(resUrl); cacheRes != nil {
		return
	}

	if res := fetch(resUrl); res != nil {
		cacheRes = mkCacheFrom(resUrl, res)
	}

	return
}

var fetch = func(resUrl string) (cache *http.Response) {
	host, _ := url.Parse(resUrl)
	req, err := http.NewRequest(http.MethodGet, resUrl, nil)
	if err != nil {
		log.Fatalf("Something went wrong while creating a new request. (err: %v)", err)
	}

	req.Header.Add("Content-Type", "text/plain")
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Accept-Language", "en-US,en;q=0.7,ja;q=0.3")
	req.Header.Add("Host", host.Host)
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:103.0) Gecko/20100101 Firefox/103.0")
	req.Header.Add("Sec-Fetch-Dest", "document")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("Sec-Fetch-Site", "none")
	req.Header.Add("Sec-Fetch-User", "?1")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	req.Header.Add("Connection", "keep-alive")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Something went wrong while fetch the request. (err: %v)", err)
	}

	return res
}

var mkCacheFrom = func(resUrl string, res *http.Response) (cache *CacheResponse) {
	sum := getSha512Sum(resUrl)
	cachePath := savedCachePath + sum

	cacheContent, err := os.Create(cachePath + ".cache")
	if err != nil {
		log.Fatalf("error: cannot create cache (%v)", err)
	}

	cacheFile, err := os.Create(cachePath + ".json")
	if err != nil {
		log.Fatalf("error: cannot create file (%v)", err)
	}

	cache = &CacheResponse{
		Cache_path: cachePath,
		E_at:       time.Now().Add(time.Minute * 4),
		Response:   res,
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("%v (output: %v)", err, resBody)
	}

	res.Body.Close()

	cache.Body = nil
	cache.Request = nil
	cache.TLS = nil

	jsonBody, err := json.Marshal(cache)
	if err != nil {
		log.Fatalf("%v (output: %v)", err, jsonBody)
	}

	cacheFile.Write(jsonBody)
	cacheFile.Close()

	cacheContent.Write(resBody)
	cacheContent.Close()

	return getCache(resUrl)
}

var getCache = func(resUrl string) (cache *CacheResponse) {
	key := getSha512Sum(resUrl)
	cachePath := savedCachePath + key

	P, err := ioutil.ReadFile(cachePath + ".json")
	if err != nil {
		return nil
	}

	cache = &CacheResponse{}

	err = json.Unmarshal(P, cache)
	if err != nil {
		log.Fatalf("cannot unmarshal cache. (error: %v)", err)
	}

	cacheContent, err := os.Open(cachePath + ".cache")
	if err != nil {
		log.Fatalf("cannot read content of the cache. (error: %v)", err)
	}

	cache.Body = cacheContent
	return
}
