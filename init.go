package dogfetch

import (
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/rommms07/dogfetch/internal/utils"
)

var (
	mu sync.Mutex
	wg sync.WaitGroup

	// Usually making a series of concurrent HTTP request into the server causes an Internal Server Error
	// on the server, to avoid that issue, we limit the number of concurrent request up to N by utilizing the
	// properties of bufferred channels. In this case we limit the number of parallel HTTP request by
	// 50.
	queue       = make(chan string, 50)
	fetchResult = make(map[string]*BreedInfo)
	isTesting   = regexp.MustCompile(`^-test(.+)$`).MatchString(os.Args[1])
)

func fetchDogBreeds() (dogs map[string]*BreedInfo) {
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
		queue <- pagePath
		wg.Add(1)
		go crawlPage(pagePath)
	}

	wg.Wait()
	dogs = fetchResult
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
	sum := utils.GetMd5Sum(path)
	fetchResult[sum] = digPage(P)

	// After digging all information from various resources, we add the default resource into the
	// references field of the object.
	fetchResult[sum].Id = sum
	fetchResult[sum].Refs = append(fetchResult[sum].Refs, string(Source1)+path)
	mu.Unlock()

	wg.Done()
	<-queue
}

func digPage(P []byte) (bi *BreedInfo) {
	bi = &BreedInfo{}

	mainFmt := `(?m)(?s)<div class="content">(.|\n)*?`
	namePatt := regexp.MustCompile(mainFmt + `<h1>(?P<name>[^<]+)<\/h1>`)
	refsPatt := regexp.MustCompile(mainFmt + `(<div class="like">|<h3>References</h3>)(?P<url>.+)<\/ul>`)
	otherNamesPatt := regexp.MustCompile(mainFmt + `<td>Other names<\/td>.*?<td>(?P<otherNames>[^<]+?)<\/td>`)
	breedGroupsPatt := regexp.MustCompile(mainFmt + `<td>Breed Group<\/td>.*?<td>(?P<breedGroups>.+?)<\/td>`)
	originPatt := regexp.MustCompile(mainFmt + `<td>Origin<\/td>.*?<td class\="flag">(?P<origin>.+?)(<\/td>)`)
	sizePatt := regexp.MustCompile(mainFmt + `<td>Size</td>.*?<td>(?P<size>.+?)<\/td>`)

	indices := namePatt.FindSubmatchIndex(P)
	bi.Name = string(namePatt.Expand([]byte{}, []byte(`$name`), P, indices))

	indices = refsPatt.FindSubmatchIndex(P)
	refs := refsPatt.Expand([]byte{}, []byte(`$url`), P, indices)
	hrefPatt := regexp.MustCompile(`href="(?P<url>.+?)"`)
	for _, indices := range hrefPatt.FindAllSubmatchIndex(refs, -1) {
		href := string(hrefPatt.Expand([]byte{}, []byte(`$url`), refs, indices))
		bi.Refs = append(bi.Refs, string(Source1)+href)

		if strings.Contains(href, "//") {
			href = regexp.MustCompile(`^//`).ReplaceAllString(href, "https://")
			continue
		}

		bi.BreedRecs = append(bi.BreedRecs, utils.GetMd5Sum(href))
	}

	for _, indices := range otherNamesPatt.FindAllSubmatchIndex(P, -1) {
		otherNames := string(otherNamesPatt.Expand([]byte{}, []byte(`$otherNames`), P, indices))
		bi.OtherNames = append(bi.OtherNames, strings.Split(otherNames, ",")...)
		bi.OtherNames = cleanStrings(bi.OtherNames)
	}

	indices = originPatt.FindSubmatchIndex(P)
	origins := string(originPatt.Expand([]byte{}, []byte(`$origin`), P, indices))
	origins = removeMisc(origins)
	bi.Origin = cleanResults(strings.Split(origins, "</p>"))

	indices = breedGroupsPatt.FindSubmatchIndex(P)
	breedGroups := string(breedGroupsPatt.Expand([]byte{}, []byte(`$breedGroups`), P, indices))
	breedGroups = removeMisc(breedGroups)
	bi.BreedGroups = cleanResults(strings.Split(breedGroups, "</p>"))

	indices = sizePatt.FindSubmatchIndex(P)
	size := string(sizePatt.Expand([]byte{}, []byte(`$size`), P, indices))
	size = removeMisc(size)
	bi.Size = cleanResults(strings.Split(size, "to"))
	return
}

func cleanResults(S []string) (s []string) {
	for _, str := range S {
		if len(str) == 0 {
			continue
		}

		splitCon := strings.Split(regexp.MustCompile(`\([^(]+\)`).ReplaceAllString(replaceConjunctions(str), ""), " and ")
		s = append(s, splitCon...)
	}

	return cleanStrings(s)
}

func replaceConjunctions(S string) string {
	// and
	res := regexp.MustCompile(`(\\[u][0]{2}26amp\;|&)`).ReplaceAllString(S, "and")
	res = strings.ReplaceAll(res, "amp;", "")

	return res
}

func cleanStrings(S []string) []string {
	for i, name := range S {
		S[i] = strings.TrimSpace(name)
		S[i] = strings.ReplaceAll(S[i], "&#39;", "'")
	}

	return S
}

func removeMisc(s string) string {
	return regexp.MustCompile(`(<img .*?>\s|&nbsp;|<p>|\n|\r|<a .*?>|</a>)`).ReplaceAllString(s, "")
}
