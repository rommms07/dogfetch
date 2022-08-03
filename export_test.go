package dogfetch

var (
	// Since the sync.WaitGroup object in init.go is defined by value, we cannot modify
	// its state directly from the test code. So we assign its reference value into this
	// test exported identifier to refer to it later in the test.
	WGroup       = &wg
	Queue        = queue
	FetchResults = fetchResult

	FetchDogBreeds = fetchDogBreeds

	// crawlPage is tightly cooupled with the digPage function, so if we are testing
	// this unexported we may in turn also be testing the digPage function. Hence it is not
	// necessary to export the digPage function for testing because of this relationship.
	CrawlPage = crawlPage
)
