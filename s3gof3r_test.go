package s3gof3r

import (
	"testing"
)

type uploadTest struct {
	url       string
	file_path string
	header    string
	check     bool
	out       string
}

type downloadTest struct {
	url       string
	file_path string
	out       string
}

var uploadTests = []uploadTest{
	{"https://foo", "bar_file", "a:b", false, "out"},
}

var downloadTests = []downloadTest{
	{"https://foo", "bar_file", "out"},
}

func TestUpload(t *testing.T) {
	for _, tt := range uploadTests {
		actual := ""
		if actual != tt.out {
			t.Errorf("Upload called with %s, %s, %s. Expected %s, actual %s",
				tt.url, tt.file_path, tt.header, tt.out, actual)

		}
	}
}

func TestDownload(t *testing.T) {
	for _, tt := range downloadTests {
		actual := ""
		if actual != tt.out {
			t.Errorf("Download called with %s, %s. Expected %s, actual %s",
				tt.url, tt.file_path, tt.out, actual)

		}
	}
}
