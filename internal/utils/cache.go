package utils

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
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
	req, err := http.NewRequest(http.MethodGet, resUrl, nil)
	if err != nil {
		log.Fatalf("Something went wrong while creating a new request. (err: %v)", err)
	}

	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:103.0) Gecko/20100101 Firefox/103.0")

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
