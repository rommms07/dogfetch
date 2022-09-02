package dogfetch

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	URL "net/url"
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
	getReferencesData(fetchResult[sum], []string{string(Source1) + path})
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
	litterSizePatt := regexp.MustCompile(mainFmt + `<td>Litter Size<\/td>.*?<td>(?P<start>[\d]+?)-(?P<end>[\d]+?).+?<\/td>`)
	historyPatt := regexp.MustCompile(mainFmt + `<h2>History<\/h2>.*?<td>.*?<p>(?P<history>.+?)<\/p>`)

	indices := namePatt.FindSubmatchIndex(P)
	bi.Name = string(namePatt.Expand([]byte{}, []byte(`$name`), P, indices))

	indices = typePatt.FindSubmatchIndex(P)
	bi.Type = string(typePatt.Expand([]byte{}, []byte(`$type`), P, indices))

	indices = refsPatt.FindSubmatchIndex(P)
	refs := refsPatt.Expand([]byte{}, []byte(`$url`), P, indices)
	hrefPatt := regexp.MustCompile(`href="(?P<url>.+?)"`)
	urls := []string{}
	for _, indices := range hrefPatt.FindAllSubmatchIndex(refs, -1) {
		href := string(hrefPatt.Expand([]byte{}, []byte(`$url`), refs, indices))

		if strings.Contains(href, "//") {
			href = regexp.MustCompile(`^//`).ReplaceAllString(href, "https://")
			urls = append(urls, href)
			continue
		}

		urls = append(urls, string(Source1)+href)
		bi.BreedRecs = append(bi.BreedRecs, utils.GetMd5Sum(href))
	}

	bi.Refs = make(map[string]any)
	getReferencesData(bi, urls)

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

func getReferencesData(bi *BreedInfo, urls []string) {
	youtubeUrlPatt := regexp.MustCompile("youtube\\.com")
	pdfPatt := regexp.MustCompile("\\.pdf")

	for _, href := range urls {
		wg.Add(1)
		(func(bi *BreedInfo, href string) {
			var data any
			var res *utils.CacheResponse

			if youtubeUrlPatt.MatchString(href) {
				res, _ = utils.NewCacheResponse("https://www.youtube.com/oembed?url=" + href)
				P, _ := io.ReadAll(res.Body)
				data = make(map[string]any)

				json.Unmarshal(P, &data)
			} else if pdfPatt.MatchString(href) {
				data = href
			} else {
				res, _ = utils.NewCacheResponse(href)
				P, _ := io.ReadAll(res.Body)
				metaData := make(map[string]any)
				isSpecialCase := false

				ogMetaPatt := []string{
					`(<meta.*?name\="?twitter:title"?.*?content\="(?P<title>[^/>]+?)".*?>|<meta.*?property\="?og:title"?.*?content\="(?P<title>[^/>]+?)".*?>|<title>(?P<title>[^/]+)<\/title>)`,
					`(<meta.*?property\="?twitter:description"?.*?content\="(?P<desc>[^/>]?).*?>|<meta.*?property\="?og:description"?.*?content\="(?P<desc>[^/>]+?)".*?>|<meta.*?name\="description".*?content\="(?P<desc>[^/]+?)".*?>|<h3>一般外貌<\/h3>.*?<dd>.*?<p>(?P<desc>[^/>]+?)<\/p>.*?<\/dd>)`,
				}

				oGraphPatt := regexp.MustCompile("(?s)" + strings.Join(ogMetaPatt, ".+?"))
				mainSitePatt := regexp.MustCompile("(dogbreedslist\\.info|www\\.wikihow\\.com)")

				indices := oGraphPatt.FindSubmatchIndex(P)
				oGraphDataBs := oGraphPatt.Expand([]byte{}, []byte(`$title@#$desc@#$title@#$ogUrl@#$ogSiteName`), P, indices)
				oGraphData := strings.Split(string(oGraphDataBs), "@#")

				if len(strings.TrimSpace(strings.Join(oGraphData, ""))) != 0 {
					metaData["title"] = oGraphData[0]
					metaData["description"] = oGraphData[1]
				} else {
					isSpecialCase = true
				}

				if !mainSitePatt.Match(P) {
					phref, _ := URL.Parse(href)
					jpegRef := regexp.MustCompile(`<img.*?(loading="lazy".*?data-src\="(?P<image>https?://.*?\.jpg)\"|src\="(?P<image>(.*?\/img\/breeds\/[^/>"]*)[^/>"]*?)").*?>`)
					indices := jpegRef.FindAllSubmatchIndex(P, -1)

					images := make(map[string]bool)
					for _, index := range indices {
						matchBs := jpegRef.Expand([]byte{}, []byte("$image"), P, index)

						if !images[string(matchBs)] {
							images[string(matchBs)] = true
						}
					}

					for image := range images {
						if regexp.MustCompile(`.*?Danish.*?`).MatchString(image) {
							continue
						}

						if !regexp.MustCompile(`https?:\/\/`).MatchString(image) {
							bi.Images = append(bi.Images, fmt.Sprintf("%s://%s%s", phref.Scheme, phref.Host, image[0:]))
							continue
						}

						bi.Images = append(bi.Images, image)
					}
				}

				if !isSpecialCase {
					data = metaData
				} else {
					data = href
				}
			}

			if res != nil {
				defer res.Body.Close()
			}

			bi.Refs[href] = data
			wg.Done()
		})(bi, href)
	}

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
