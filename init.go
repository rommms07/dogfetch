package dogfetch

import (
	"io"
	"log"
	"os"
	"regexp"
	"sync"

	"github.com/rommms07/dogfetch/internal/utils"
)

var (
	mu          sync.Mutex
	wg          sync.WaitGroup
	fetchResult = []string{}
	isTesting   = regexp.MustCompile(`^-test(.+)$`).MatchString(os.Args[1])
)

func fetchDogBreeds() (dogs []string) {
	/**

	 */
	const (
		listBreeds = string(Source1) + "/dog-breeds-a-z/"
		s2Search   = string(Source2) + "/search/"
		s3Search   = string(Source3) + "/search/content-search"
	)

	patt := regexp.MustCompile(`<dd><a href="(?P<page>/all-dog-breeds/[^.]+.html)">.+?</a></dd>`)

	pageListRes, _ := utils.NewCacheResponse(listBreeds)

	defer pageListRes.Body.Close()
	P, err := io.ReadAll(pageListRes.Body)
	if err != nil {
		log.Fatalf("error reading bytes (err: %v)", err)
	}

	for _, indexes := range patt.FindAllSubmatchIndex(P, -1) {
		pagePath := string(patt.Expand([]byte{}, []byte(`$page`), P, indexes))

		wg.Add(1)
		go crawlPage(pagePath)
	}

	wg.Wait()
	dogs = append(dogs, fetchResult...)
	return
}

func crawlPage(path string) {
	breedp, _ := utils.NewCacheResponse(string(Source1) + path)
	defer breedp.Body.Close()

	P, err := io.ReadAll(breedp.Body)
	if err != nil {
		log.Fatalf("error reading bytes (err: %v)", err)
	}

	mu.Lock()
	fetchResult = append(fetchResult, string(P))
	mu.Unlock()

	wg.Done()

}
