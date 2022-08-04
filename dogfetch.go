package dogfetch

import "sync"

type BreedInfo struct {
	Id           string           `json:"id"`
	Type         string           `json:"type"`
	Name         string           `json:"name"`
	Size         []string         `json:"size"`
	Origin       []string         `json:"origins"`
	Colors       []string         `json:"colors"`
	Images       []string         `json:"images"`
	Temperaments []string         `json:"temperaments"`
	OtherNames   []string         `json:"otherNames"`
	BreedGroups  []string         `json:"breedGroups"`
	BreedChars   map[string]int64 `json:"breedChars"`
	BreedRecs    []string         `json:"breedRecs"`
	Refs         []string         `json:"refs"`
}

type BreedInfos map[string]*BreedInfo

func (bis BreedInfos) GetByName(name string) (res *BreedInfo) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, bi := range bis {

		wg.Add(1)
		go func(bi *BreedInfo) {
			if bi.Name == name {
				mu.Lock()
				res = bi
				mu.Unlock()
			}
			wg.Done()
		}(bi)
	}

	wg.Wait()
	return
}
