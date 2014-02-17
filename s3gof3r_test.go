package s3gof3r

import (
	"testing"
)

type putTest struct {
	url       string
	file_path string
	header    string
	check     bool
	out       string
}

type getTest struct {
	url       string
	file_path string
	out       string
}

var putTests = []putTest{
	{"https://foo", "bar_file", "a:b", false, "out"},
}

var getTests = []getTest{
	{"https://foo", "bar_file", "out"},
}

func TestPut(t *testing.T) {
	for _, tt := range putTests {
		actual := ""
		if actual != tt.out {
			t.Errorf("put called with %s, %s, %s. Expected %s, actual %s",
				tt.url, tt.file_path, tt.header, tt.out, actual)

		}
	}
}

func TestGet(t *testing.T) {
	for _, tt := range getTests {
		actual := ""
		if actual != tt.out {
			t.Errorf("get called with %s, %s. Expected %s, actual %s",
				tt.url, tt.file_path, tt.out, actual)

		}
	}
}
