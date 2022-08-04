package dogfetch_test

import (
	"net/url"
	"testing"

	"github.com/rommms07/dogfetch"
	"github.com/rommms07/dogfetch/internal/utils"
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

var expectedResults = []expTestResults{
	{
		breedInfo: &dogfetch.BreedInfo{
			Name: "Australian Shepherd",
			Refs: []string{
				"https://www.dogbreedslist.info/all-dog-breeds/miniature-american-shepherd.html",
				"https://www.dogbreedslist.info/all-dog-breeds/border-collie.html",
				"https://www.dogbreedslist.info/all-dog-breeds/shetland-sheepdog.html",
				"https://www.dogbreedslist.info/all-dog-breeds/australian-cattle-dog.html",
			},
			OtherNames: []string{
				"Aussie",
				"Little Blue Dog",
			},
		},
	},
	{
		breedInfo: &dogfetch.BreedInfo{
			Name: "American Cocker Spaniel",
			Refs: []string{
				"https://www.dogbreedslist.info/all-dog-breeds/english-cocker-spaniel.html",
				"https://www.dogbreedslist.info/all-dog-breeds/english-springer-spaniel.html",
				"https://www.dogbreedslist.info/all-dog-breeds/cavalier-king-charles-spaniel.html",
				"https://www.dogbreedslist.info/all-dog-breeds/cockapoo.html",
			},
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
			Refs: []string{
				"https://www.dogbreedslist.info/all-dog-breeds/cockapoo.html",
				"https://www.dogbreedslist.info/all-dog-breeds/argentine-dogo.html",
				"https://www.dogbreedslist.info/all-dog-breeds/rottweiler.html",
				"https://www.dogbreedslist.info/all-dog-breeds/american-akita.html",
			},
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
			Refs: []string{
				"https://www.dogbreedslist.info/all-dog-breeds/bull-arab.html",
				"https://www.dogbreedslist.info/all-dog-breeds/scottish-deerhound.html",
				"https://www.dogbreedslist.info/all-dog-breeds/irish-wolfhound.html",
				"https://www.dogbreedslist.info/all-dog-breeds/catahoula-leopard-dog.html",
			},
			OtherNames: []string{
				"Staghound",
			},
		},
	},
	{
		breedInfo: &dogfetch.BreedInfo{
			Name: "Yorkshire Terrier",
			Refs: []string{
				"https://www.dogbreedslist.info/all-dog-breeds/shih-tzu.html",
				"https://www.dogbreedslist.info/all-dog-breeds/maltese-dog.html",
				"https://www.dogbreedslist.info/all-dog-breeds/chihuahua.html",
				"https://www.dogbreedslist.info/all-dog-breeds/pomeranian.html",
			},
			OtherNames: []string{
				"Yorkie",
			},
		},
	},
}

func Test_fetchDogBreeds(t *testing.T) {
	const EXPECTED_NUM_DOGS = 373

	dogs := dogfetch.FetchDogBreeds()

	if len(dogs) != EXPECTED_NUM_DOGS {
		t.Errorf("(fail) Expected number of dogs did not matched! (expected: %d)", EXPECTED_NUM_DOGS)
	}
}

func Test_crawlPage(t *testing.T) {
	for i, T := range testUrls {
		resUrl, err := url.Parse(T)
		if err != nil {
			t.Errorf("(fail) Unable to parse the url: %s", T)
		}

		path := resUrl.Path
		sum := utils.GetMd5Sum(path)

		dogfetch.Queue <- path
		dogfetch.WGroup.Add(1)
		go dogfetch.CrawlPage(path)

		dogfetch.WGroup.Wait()

		res := dogfetch.FetchResults[sum]
		expect := expectedResults[i]

		if res.Name != expect.breedInfo.Name {
			t.Errorf("(fail) Did not matched the expected dog breed name. (%s != %s)", res.Name,
				expect.breedInfo.Name)
		}

		var refs int

		// Check if the URL references field contains the tested URL above.
		// if not we issue an assertion error.
		for _, ref := range res.Refs {
			for _, expRef := range expect.breedInfo.Refs {
				if expRef == ref {
					refs++
				}
			}
		}

		if float64(refs)/float64(len(res.Refs)) < 0.5 {
			t.Errorf("(fail) Digged reference is incomplete, verify it!")
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

func Test_breedInfos_GetByName(t *testing.T) {
	for _, T := range expectedResults {
		if dogfetch.FetchResults.GetByName(T.breedInfo.Name).Name != T.breedInfo.Name {
			t.Errorf("(fail) Did not matched expected dog breed name! (input: %v)", T.breedInfo.Name)
		}
	}
}
