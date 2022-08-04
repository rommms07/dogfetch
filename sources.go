package dogfetch

type BreedSource string

const (
	Source1 BreedSource = "https://www.dogbreedslist.info"
	Source2             = "http://www.infodogs.co.uk"
	Source3             = "https://www.thekennelclub.org.uk"
)

type BreedInfo struct {
	Id           string
	Type         string
	Name         string
	Size         []string
	Origin       []string
	Colors       []string
	Images       []string
	Temperaments []string
	OtherNames   []string
	BreedGroups  []string
	BreedChars   map[string]int
	BreedRecs    []string
	Refs         []string
}
