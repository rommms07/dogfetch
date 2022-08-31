package dogfetch

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
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
	fetchResult = make(BreedInfos)
)

func fetchDogBreeds() (dogs map[string]*BreedInfo) {
	if P, err := ioutil.ReadFile("/tmp/breeds.json"); err == nil {
		err := json.Unmarshal(P, &fetchResult)
		if err != nil {
			log.Fatal(err)
		}

		dogs = fetchResult
		return
	}

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

	if P, err := json.Marshal(dogs); err != nil {
		err := ioutil.WriteFile("/tmp/breeds.json", P, 0660)
		if err != nil {
			log.Fatal(err)
		}
	}
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
	bi = &BreedInfo{
		BreedChars: make(map[string]int64),
	}

	mainFmt := `(?m)(?s)<div class="content">(.|\n)*?`
	namePatt := regexp.MustCompile(mainFmt + `<h1>(?P<name>[^<]+)<\/h1>`)
	imagesPatt := regexp.MustCompile(mainFmt + `<div class="slideshow">(?P<images>.+?)<table class="table-03">.*?`)
	refsPatt := regexp.MustCompile(mainFmt + `(<div class="like">|<h3>References</h3>)(?P<url>.+)<\/ul>`)
	otherNamesPatt := regexp.MustCompile(mainFmt + `<td>Other names<\/td>.*?<td>(?P<otherNames>[^<]+?)<\/td>`)
	breedGroupsPatt := regexp.MustCompile(mainFmt + `<td>Breed Group<\/td>.*?<td>(?P<breedGroups>.+?)<\/td>`)
	originPatt := regexp.MustCompile(mainFmt + `<td>Origin<\/td>.*?<td class\="flag">(?P<origin>.+?)(<\/td>)`)
	sizePatt := regexp.MustCompile(mainFmt + `<td>Size</td>.*?<td>(?P<size>.+?)<\/td>`)
	tempPatt := regexp.MustCompile(mainFmt + `<td>Temperament<\/td>.*?<td>*(?P<temperaments>.+?)<\/td>`)
	colorsPatt := regexp.MustCompile(mainFmt + `<td>Colors<\/td>.*?<td>*(?P<colors>.+?)<\/td>`)
	typePatt := regexp.MustCompile(mainFmt + `<td>Type<\/td>.*?<td>(?P<type>.+?)<\/td>`)
	charsPatt := regexp.MustCompile(mainFmt + `<table class="table-02">.*?<tbody>.*Breed Characteristics.*?(?P<chars>.+?)<\/tbody>.*?<\/table>`)
	lspanPatt := regexp.MustCompile(mainFmt + `<td>Life span<\/td>.*?<td>(?P<start>[\d]+?)-(?P<end>[\d]+?).+?<\/td>`)
	litterSizePatt := regexp.MustCompile(mainFmt + `<td>Litter Size<\/td>.*?<td>(?P<start>[\d]+?)-(?P<end>[\d]+?)<\/td>`)
	historyPatt := regexp.MustCompile(mainFmt + `<h2>History<\/h2>.*?<div class\="fold-text">.*?<p>(?P<history>.+?)<\/p>`)
	// heightPatt := regexp.MustCompile(mainFmt + ``)
	// weightPatt := regexp.MustCompile(mainFmt + ``)

	indices := namePatt.FindSubmatchIndex(P)
	bi.Name = string(namePatt.Expand([]byte{}, []byte(`$name`), P, indices))

	indices = typePatt.FindSubmatchIndex(P)
	bi.Type = string(typePatt.Expand([]byte{}, []byte(`$type`), P, indices))

	indices = refsPatt.FindSubmatchIndex(P)
	refs := refsPatt.Expand([]byte{}, []byte(`$url`), P, indices)
	hrefPatt := regexp.MustCompile(`href="(?P<url>.+?)"`)
	for _, indices := range hrefPatt.FindAllSubmatchIndex(refs, -1) {
		href := string(hrefPatt.Expand([]byte{}, []byte(`$url`), refs, indices))

		if strings.Contains(href, "//") {
			href = regexp.MustCompile(`^//`).ReplaceAllString(href, "https://")
			bi.Refs = append(bi.Refs, href)
			continue
		}

		bi.Refs = append(bi.Refs, string(Source1)+href)
		bi.BreedRecs = append(bi.BreedRecs, utils.GetMd5Sum(href))
	}

	for _, indices := range otherNamesPatt.FindAllSubmatchIndex(P, -1) {
		otherNames := string(otherNamesPatt.Expand([]byte{}, []byte(`$otherNames`), P, indices))
		bi.OtherNames = append(bi.OtherNames, strings.Split(otherNames, ",")...)
		bi.OtherNames = cleanStrings(bi.OtherNames)
	}

	bi.OtherNames = uniqueSet(bi.OtherNames)

	bi.Origin = getResults(originPatt, "</p>", []byte(`$origin`), P)
	bi.BreedGroups = getResults(breedGroupsPatt, "</p>", []byte(`$breedGroups`), P)
	bi.Size = getResults(sizePatt, "to", []byte(`$size`), P)
	bi.Temperaments = getResults(tempPatt, "</p>", []byte(`$temperaments`), P)
	bi.Colors = func() []string {
		results := getResults(colorsPatt, "</p>", []byte(`$colors`), P)
		maps := make(map[string]int)
		for _, val := range results {
			maps[val] = 1
		}

		results = make([]string, 0)

		for k := range maps {
			results = append(results, k)
		}

		return results
	}()

	indices = charsPatt.FindSubmatchIndex(P)
	chars := charsPatt.Expand([]byte{}, []byte(`$chars`), P, indices)
	charsTypePatt := regexp.MustCompile(`(?m)<td>(?P<type>[A-Za-z ]*?)</td>(.|\n)*?<p class="star-0\d">(?P<score>\d) stars<\/p>`)

	for _, indices := range charsTypePatt.FindAllSubmatchIndex(chars, -1) {
		chars := strings.Split(string(charsTypePatt.Expand([]byte{}, []byte(`$type,$score`), chars, indices)), ",")

		if len(chars[0]) == 0 {
			continue
		}

		score, err := strconv.ParseInt(chars[1], 10, 64)
		if err != nil {
			score = 0
		}

		bi.BreedChars[strings.TrimSpace(chars[0])] = score
	}

	indices = imagesPatt.FindSubmatchIndex(P)
	images := imagesPatt.Expand([]byte{}, []byte(`$images`), P, indices)
	srcPatt := regexp.MustCompile(`<img.*?src="(?P<imageSrc>\/uploads\/dog-pictures\/[^"]+)"`)

	for _, indices := range srcPatt.FindAllSubmatchIndex(images, -1) {
		src := string(srcPatt.Expand([]byte{}, []byte(`$imageSrc`), images, indices))
		bi.Images = append(bi.Images, string(Source1)+src)
	}

	indices = lspanPatt.FindSubmatchIndex(P)
	lifespan := lspanPatt.Expand([]byte{}, []byte(`$start-$end`), P, indices)

	for _, years := range strings.Split(string(lifespan), "-") {
		y, _ := strconv.ParseUint(years, 10, 64)
		bi.Lifespan = append(bi.Lifespan, y)
	}

	indices = litterSizePatt.FindSubmatchIndex(P)
	litterSize := litterSizePatt.Expand([]byte{}, []byte(`$start-$end`), P, indices)

	for _, litter := range strings.Split(string(litterSize), "-") {
		l, _ := strconv.ParseUint(litter, 10, 64)
		bi.LitterSize = append(bi.LitterSize, l)
	}

	indices = historyPatt.FindSubmatchIndex(P)
	history := historyPatt.Expand([]byte{}, []byte(`$history`), P, indices)

	bi.History = string(history)

	return
}

func getResults(patt *regexp.Regexp, sep string, tmp, P []byte) []string {
	indices := patt.FindSubmatchIndex(P)
	results := string(patt.Expand([]byte{}, []byte(tmp), P, indices))
	results = removeMisc(results)
	return cleanResults(strings.Split(results, sep))
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

func uniqueSet(sub []string) []string {
	var M = make(map[string]bool)
	var res = make([]string, 0)

	for _, name := range sub {
		M[name] = true
	}

	for name := range M {
		res = append(res, name)
	}

	return res
}
