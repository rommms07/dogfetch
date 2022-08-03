package dogfetch_test

import (
	"net/url"
	"testing"

	"github.com/rommms07/dogfetch"
)

type expTestResults struct {
	breedInfo *dogfetch.BreedInfo
}

var (
	testUrls = []string{
		"https://www.dogbreedslist.info/all-dog-breeds/australian-shepherd.html",
		"https://www.dogbreedslist.info/all-dog-breeds/american-cocker-spaniel.html",
		"https://www.dogbreedslist.info/all-dog-breeds/alano-espanol.html",
		"https://www.dogbreedslist.info/all-dog-breeds/american-staghound.html",
	}
)

func Test_fetchDogBreeds(t *testing.T) {
	const EXPECTED_NUM_DOGS = 373

	dogs := dogfetch.FetchDogBreeds()

	if len(dogs) != EXPECTED_NUM_DOGS {
		t.Errorf("(fail) Expected number of dogs did not matched! (expected: %d)", EXPECTED_NUM_DOGS)
	}
}

func Test_crawlPage(t *testing.T) {
	var expectedResults = []expTestResults{
		{
			breedInfo: &dogfetch.BreedInfo{
				Name: "Australian Shepherd",
				OtherNames: []string{
					"Aussie",
					"Little Blue Dog",
				},
			},
		},
		{
			breedInfo: &dogfetch.BreedInfo{
				Name: "American Cocker Spaniel",
				OtherNames: []string{
					"Cocker Spaniel",
					"Cocker",
					"Merry Cocker",
				},
			},
		},
		{
			breedInfo: &dogfetch.BreedInfo{
				Name: "Alano Espanol",
				OtherNames: []string{
					"Spanish Alano",
					"Spanish Bulldog",
					"Alano",
				},
			},
		},
		{
			breedInfo: &dogfetch.BreedInfo{
				Name: "American Staghound",
				OtherNames: []string{
					"Staghound",
				},
			},
		},
	}

	for i, T := range testUrls {
		resUrl, err := url.Parse(T)
		if err != nil {
			t.Errorf("(fail) Unable to parse the url: %s", T)
		}

		path := resUrl.Path

		dogfetch.Queue <- path
		dogfetch.WGroup.Add(1)
		go dogfetch.CrawlPage(path)

		dogfetch.WGroup.Wait()

		res := dogfetch.FetchResults[path]
		expect := expectedResults[i]

		if res.Name != expect.breedInfo.Name {
			t.Errorf("(fail) Did not matched the expected dog breed name. (%s != %s)", res.Name,
				expect.breedInfo.Name)
		}

		// Check if the URL references field contains the tested URL above.
		// if not we issue an assertion error.
		var hasRef bool

		for _, ref := range res.Refs {
			if ref == T {
				hasRef = true
			}
		}

		if !hasRef {
			t.Errorf("(fail) References did not contain the tested URL. (%s)", T)
		}

		for _, expectName := range expect.breedInfo.OtherNames {
			contains := false

			for _, otherName := range res.OtherNames {
				if expectName == otherName {
					contains = true
				}
			}

			if !contains {
				t.Errorf("(fail) Did not contain one of the expected other names of the breed. ('%s' missing)", expectName)
			}
		}
	}
}
