package dogfetch

import (
	"sync"
)

type BreedInfo struct {
	Id           string           `json:"id"`
	History      string           `json:"history"`
	Type         string           `json:"type"`
	Name         string           `json:"name"`
	Size         []string         `json:"size"`
	Origin       []string         `json:"origins"`
	Colors       []string         `json:"colors"`
	Images       []string         `json:"images"`
	Lifespan     []uint64         `json:"lifeSpan"`
	LitterSize   []uint64         `json:"litterSize"`
	Temperaments []string         `json:"temperaments"`
	OtherNames   []string         `json:"otherNames"`
	BreedGroups  []string         `json:"breedGroups"`
	BreedChars   map[string]int64 `json:"breedChars"`
	BreedRecs    []string         `json:"breedRecs"`
	Refs         map[string]any   `json:"refs"`
}

type Refs = map[string]map[string]string

func init() {
	fetchDogBreeds()
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

func GetById(id string) *BreedInfo {
	return fetchResult[id]
}

func GetByName(name string) (res *BreedInfo) {
	return fetchResult.GetByName(name)
}

func GetAll() BreedInfos {
	return fetchResult
}
