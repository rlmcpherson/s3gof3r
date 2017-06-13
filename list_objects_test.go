package s3gof3r

import (
	"log"
	"sort"
	"sync"
	"testing"
)

var keysForListing = []string{
	"list/one/two/three",
	"list/one/two/four",
	"list/two/three/four",
	"list/two/three/five",
	"list/three/four/five",
	"list/three/four/six",
	"list/four/five/six",
	"list/four/five/seven",
}

func uploadListerFiles() {
	var wg sync.WaitGroup
	for _, tt := range keysForListing {
		wg.Add(1)
		go func(path string) {
			err := b.putReader(path, &randSrc{Size: 20})
			if err != nil {
				log.Fatal(err)
			}
			wg.Done()
		}(tt)
	}
	wg.Wait()
}

func testListObjects(t *testing.T, prefixes []string, iterations, concurrency int) {
	config := Config{
		Concurrency: 1,
		Scheme:      "https",
	}
	l, err := b.ListObjects(prefixes, 5, &config)
	if err != nil {
		t.Error(err)
	}

	actual := make([]string, 0, len(keysForListing))
	actualIterations := 0
	for l.Next() {
		actualIterations++
		actual = append(actual, l.Value()...)
	}

	err = l.Error()
	if err != nil {
		t.Error(err)
	}

	if actualIterations != iterations {
		t.Errorf("expected %d iterations, got %d", iterations, actualIterations)
	}

	if len(actual) != len(keysForListing) {
		t.Errorf("expected %d keys, got %d", len(keysForListing), len(actual))
	}

	sort.Strings(keysForListing)
	sort.Strings(actual)

	for i, a := range keysForListing {
		if a != actual[i] {
			t.Errorf("result mismatch, expected '%s', got '%s'", a, actual[i])
		}
	}
}

func TestListObjects(t *testing.T) {
	t.Parallel()

	uploadListerFiles()

	testListObjects(t, []string{"list/"}, 2, 1)
	testListObjects(t, []string{"list/"}, 2, 5)
	testListObjects(t, []string{"list/one/", "list/two/", "list/three", "list/four"}, 4, 1)
	testListObjects(t, []string{"list/one/", "list/two/", "list/three", "list/four"}, 4, 5)
}
