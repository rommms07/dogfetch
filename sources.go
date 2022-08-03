package dogfetch

type BreedSource string

const (
	Source1 BreedSource = "https://www.dogbreedslist.info"
	Source2             = "http://www.infodogs.co.uk"
	Source3             = "https://www.thekennelclub.org.uk"
)

type BreedInfo struct {
	Id           int
	Name         string
	Size         string
	Colors       []string
	Temperaments []string
	OtherNames   []string
	BreedGroups  []string
	BreedChars   map[string]int
	BreedRecs    []int
	Refs         []string
}
