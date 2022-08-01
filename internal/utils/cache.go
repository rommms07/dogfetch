package utils

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	savedCachePath = "/tmp/go-tmp/"
	inMemCacheMap  = make(map[string]*CacheResponse)
	mu             sync.Mutex
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

	if res := fetch(resUrl); res != nil {
		cacheRes = mkCacheFrom(res)
		mu.Lock()
		inMemCacheMap[key] = cacheRes
		mu.Unlock()
	}

	return
}

var fetch = func(resUrl string) *http.Response {
	if _, err := atmptLoadStoredCache(resUrl); err == nil {
		return nil
	}

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

var atmptLoadStoredCache = func(resUrl string) (*CacheResponse, error) {
	return nil, errors.New("not implemented")
}

var mkCacheFrom = func(res *http.Response) (cache *CacheResponse) {
	reqUrl := res.Request.URL.String()

	if cache = getCache(reqUrl); cache != nil && cache.E_at.UnixMilli() > time.Now().UnixMilli() {
		return
	}

	sum := getSha512Sum(reqUrl)
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

	return getCache(reqUrl)
}

var getCache = func(resUrl string) *CacheResponse {
	cachePath := savedCachePath + getSha512Sum(resUrl)

	P, err := ioutil.ReadFile(cachePath + ".json")
	if err != nil {
		return nil
	}

	cache := &CacheResponse{}

	err = json.Unmarshal(P, cache)
	if err != nil {
		log.Fatalf("cannot unmarshal cache. (error: %v)", cache)
	}

	cacheContent, err := os.Open(cachePath + ".cache")
	if err != nil {
		log.Fatalf("cannot read content of the cache. (error: %v)", err)
	}

	cache.Body = cacheContent
	return cache
}

func (res *CacheResponse) Delete() {}
