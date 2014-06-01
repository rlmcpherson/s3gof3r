package s3gof3r

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
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
	path   string
	config *Config
	rSize  int64
	err    error
}

var getTests = []getTest{
	{"testfile", nil, 22, nil},
	{"NoKey", nil, 0, &respError{StatusCode: 404, Message: "The specified key does not exist."}},
}

func TestGetReader(t *testing.T) {
	b, err := testBucket()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range getTests {
		r, h, err := b.GetReader(tt.path, tt.config)
		if !errorMatch(err, tt.err) {
			t.Errorf("GetReader called with %v\n Expected: %v\n Actual:   %v\n", tt, tt.err, err)
		}
		if err != nil {
			break
		}
		t.Logf("headers %v\n", h)
		w := ioutil.Discard

		n, err := io.Copy(w, r)
		if err != nil {
			t.Error(err)
		}
		if n != tt.rSize {
			t.Errorf("Expected size: %d. Actual: %d", tt.rSize, n)

		}
		err = r.Close()
		if err != nil {
			t.Error(err)
		}

	}
}

func testBucket() (*Bucket, error) {
	k, err := EnvKeys()
	if err != nil {
		return nil, err
	}
	bucket := os.Getenv("TEST_BUCKET")
	if bucket == "" {
		return nil, errors.New("TEST_BUCKET must be set in environment.")

	}
	s3 := New("", k)
	b := s3.Bucket(bucket)

	w, err := b.PutWriter("testfile", nil, nil)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader([]byte("Test file content....."))
	_, err = io.Copy(w, r)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}

	return b, nil

}

func errorMatch(expect, actual error) bool {

	if expect == nil && actual == nil {
		return true
	}

	if expect == nil || actual == nil {
		return false
	}

	return expect.Error() == actual.Error()

}

func ExampleBucket_PutWriter() {

	file, err := os.Open("fileName") // open file to upload
	if err != nil {
		return
	}

	k, err := EnvKeys() // get S3 keys from environment
	if err != nil {
		return
	}
	// Open bucket to put file into
	s3 := New("", k)
	b := s3.Bucket("bucketName")

	// Open a PutWriter for upload
	w, err := b.PutWriter(file.Name(), nil, nil)
	if err != nil {
		return
	}
	defer w.Close()
	if _, err = io.Copy(w, file); err != nil { // Copy into S3
		return
	}
}
