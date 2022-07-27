package utils

import (
	"context"
	"log"
	"net/http"
	"time"
)

type CacheResponse struct {
	e_at       time.Time
	cache_path string
	http.Response
}

var inMemCaches map[string]*CacheResponse

/**
 * NewCacheResponse
 *
 * Creates a new cache response by fetching the resUrl parameter value.
 */
func NewCacheResponse(resUrl string) *CacheResponse {
	if cache := fetch(resUrl); cache != nil {
		return mkCacheFrom(cache)
	}

	return getCache(resUrl)
}

func fetch(resUrl string) *http.Response {
	if err := atmptLoadStoredCache(resUrl); err == nil {
		return nil
	}

	tout := time.Second * 8
	ctx, cancel := context.WithDeadline(context.TODO(), time.Now().Add(tout))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resUrl, nil)

	defer cancel()

	if err != nil {
		log.Fatalf("Something went wrong while creating a new request. (err: %v)", err)
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Fatalf("Something went wrong while fetch the request. (err: %v)", err)
	}

	return res
}

func atmptLoadStoredCache(resUrl string) error {
	return nil
}

func mkCacheFrom(res *http.Response) *CacheResponse {
	return &CacheResponse{}
}

func getCache(resUrl string) *CacheResponse {
	return inMemCaches[getSha512Sum(resUrl)]
}

func (res *CacheResponse) Delete() {}
