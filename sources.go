package dogfetch

type BreedSource string

const (
	Source1 BreedSource = "https://www.dogbreedslist.info"
	Source2             = "http://www.infodogs.co.uk"
	Source3             = "https://www.thekennelclub.org.uk"
)

type breedInfo struct {
	id           int
	name         string
	size         string
	colors       []string
	temperaments []string
	otherNames   []string
	breedGroups  []string
	breedChars   map[string]int
	breedRecs    []int
	refs         []string
}
