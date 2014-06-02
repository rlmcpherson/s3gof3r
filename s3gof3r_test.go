package s3gof3r

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

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

type putTest struct {
	path   string
	data   []byte
	header http.Header
	config *Config
	wSize  int64
	err    error
}

var putTests = []putTest{
	{"testfile", []byte("test_data"), nil, nil, 9, nil},
	{"", []byte("test_data"), nil, nil,
		9, &respError{StatusCode: 400, Message: "A key must be specified"}},
	{"testfile", []byte(""), nil, nil, 1, nil}, //bug?
	{"testfile", []byte("foo"), correct_header(), nil, 3, nil},
}

func correct_header() http.Header {
	header := make(http.Header)
	header.Add("x-amz-server-side-encryption", "AES256")
	header.Add("x-amz-meta-foometadata", "testmeta")

	return header
}

func TestPutWriter(t *testing.T) {
	b, err := testBucket()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range putTests {
		w, err := b.PutWriter(tt.path, tt.header, tt.config)
		if !errorMatch(err, tt.err) {
			t.Errorf("PutWriter called with %v\n Expected: %v\n Actual:   %v\n", tt, tt.err, err)
		}
		if err != nil {
			break
		}
		r := bytes.NewReader(tt.data)

		n, err := io.Copy(w, r)
		if err != nil {
			t.Error(err)
		}
		if n != tt.wSize {
			t.Errorf("Expected size: %d. Actual: %d", tt.wSize, n)

		}
		err = w.Close()
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

func ExampleBucket_PutWriter() error {

	k, err := EnvKeys() // get S3 keys from environment
	if err != nil {
		return err
	}
	// Open bucket to put file into
	s3 := New("", k)
	b := s3.Bucket("bucketName")

	// open file to upload
	file, err := os.Open("fileName")
	if err != nil {
		return err
	}

	// Open a PutWriter for upload
	w, err := b.PutWriter(file.Name(), nil, nil)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, file); err != nil { // Copy into S3
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return nil
}

func ExampleBucket_GetReader() error {

	k, err := EnvKeys() // get S3 keys from environment
	if err != nil {
		return err
	}

	// Open bucket to put file into
	s3 := New("", k)
	b := s3.Bucket("bucketName")

	r, h, err := b.GetReader("keyName", nil)
	if err != nil {
		return err
	}
	// stream to standard output
	if _, err = io.Copy(os.Stdout, r); err != nil {
		return err
	}
	err = r.Close()
	if err != nil {
		return err
	}
	fmt.Println(h) // print key header data
	return nil
}
