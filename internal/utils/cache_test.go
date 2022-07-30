package utils

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
)

var testInMemCaches = make(map[string]*CacheResponse, 4)

var fakeFetch = func(resUrl string) *http.Response {
	log.Println("called fakeFetch!")

	testHdr := http.Header{}

	testHdr.Add("X-TESTING-MODE", "blahblah")
	testHdr.Add("X-TESTING-URL", resUrl)

	return &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Header:     testHdr,
	}
}

var fakeAtmptLoadStoredCache = func(resUrl string) (*CacheResponse, error) {
	log.Println("called fakeAtmptLoadStoredCache")

	if cache, exists := testInMemCaches[getSha512Sum(resUrl)]; !exists {
		return nil, errors.New("error: unable to load in-mem cache.")
	} else {
		return cache, nil
	}
}

var fakeMkCacheFrom = func(res *http.Response) *CacheResponse {
	log.Println("called fakeMkCacheFrom")

	sum := getSha512Sum(res.Header.Get("X-TESTING-URL"))

	testInMemCaches[sum] = &CacheResponse{
		E_at:       time.Now(),
		Cache_path: sum,
		Response:   res,
	}

	return testInMemCaches[sum]
}

var fakeGetCache = func(resUrl string) *CacheResponse {
	log.Println("called fakeGetCache")

	cache, _ := fakeAtmptLoadStoredCache(resUrl)
	return cache
}

func Test_NewCacheResponse(t *testing.T) {
	tests := []*TestCase{
		{
			input:    "https://www.google.com",
			expected: "20f5081d41a27c45e6cd7a7401cd97e0738a9be6ffc5897ad7d9b2dded3e4041f2208f46a2696bd86b1549f2482ebf7c4fc8cdaf9b68e454ed65b85b0dabe55b",
		},
		{
			input:    "https://pkg.go.dev/net/http#Response",
			expected: "b1a55f9d8889180abdb857601aead967d84de6f4203c163dfb43443d38d827f2bebc09789e0d09780e1da70858dec366c48683acc112602a9b78a354ace4004b",
		},
	}

	for _, T := range tests {
		unloadFakeFuncs := loadFakeFuncs()
		_, key := NewCacheResponse(T.input)

		if T.expected != key {
			t.Errorf("(fail) input: %s (expected: %s)", T.input, T.expected)
			continue
		}

		var fakeRes *CacheResponse

		// Check if the map `testInMemCaches` declared above is populated as expected.
		if testRes, exists := testInMemCaches[key]; !exists {
			t.Errorf("(fail) input: %s (did not populate testInMemCaches properly)", T.input)
			continue
		} else {
			fakeRes = testRes
		}

		if fakeRes.Status != "OK" &&
			fakeRes.StatusCode != 200 &&
			fakeRes.Header.Get("X-TESTING-MODE") != "blahblah" &&
			fakeRes.Header.Get("X-TESTING-URL") != T.input &&
			fakeRes.Cache_path != T.expected {

			t.Errorf("(fail) input: %s (did not contain the expected testing headers and response fields)", T.input)
			continue
		}

		unloadFakeFuncs()

		// Using the real implementation of fetch, we test NewCacheResponse.
		res, rkey := NewCacheResponse(T.input)

		defer res.Body.Close()

		if rkey != T.expected {
			t.Errorf("(fail) input: %s (did not match the output fake key.", T.input)
		}

		if res.Cache_path != fmt.Sprintf("/tmp/go-tmp/%s", T.expected) {
			t.Errorf("(fail) input: %s (did not match the expected path).", rkey)
		}
	}
}

func loadFakeFuncs() (unloadFakeFuncs func()) {
	var savedFetch = fetch
	var savedGetCache = getCache
	var savedAtmptLoadStoredCache = atmptLoadStoredCache
	var savedMkCacheFrom = mkCacheFrom

	// Substitute fake functions
	fetch = fakeFetch
	getCache = fakeGetCache
	atmptLoadStoredCache = fakeAtmptLoadStoredCache
	mkCacheFrom = fakeMkCacheFrom

	unloadFakeFuncs = func() {
		fetch = savedFetch
		getCache = savedGetCache
		atmptLoadStoredCache = savedAtmptLoadStoredCache
		mkCacheFrom = savedMkCacheFrom
	}

	return
}
