package dogfetch_test

import (
	"testing"

	"github.com/rommms07/dogfetch"
)

func Test_fetchDogBreeds(t *testing.T) {
	const EXPECTED_NUM_DOGS = 373

	dogs := dogfetch.FetchDogBreeds()

	if len(dogs) != EXPECTED_NUM_DOGS {
		t.Errorf("(fail) Expected number of dogs did not matched!")
	}
}
