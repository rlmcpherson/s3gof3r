package s3gof3r

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type getTest struct {
	path   string
	data   io.Reader
	config *Config
	rSize  int64
	err    error
}

var getTests = []getTest{
	{"t1.test", &randSrc{Size: int(1 * kb)}, nil, 1024, nil},
	{"NoKey", nil, nil, 0, &respError{StatusCode: 404, Message: "The specified key does not exist."}},
	{"30mb_test", &randSrc{Size: int(30 * mb)}, nil, 30 * mb, nil},
}

func TestGetReader(t *testing.T) {
	SetLogger(os.Stderr, "test: ", (log.LstdFlags | log.Lshortfile), true)
	b, err := testBucket()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range getTests {
		if tt.data != nil {
			err = b.putReader(tt.path, &tt.data)
			if err != nil {
				t.Fatal(err)
			}
		}

		r, h, err := b.GetReader(tt.path, tt.config)
		if err != nil {
			errComp(tt.err, err, t, tt)
			continue
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
		errComp(tt.err, err, t, tt)

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
	{"testempty", []byte(""), nil, nil, 0, errors.New("0 bytes written")},
	{"testhg", []byte("foo"), goodHeader(), nil, 3, nil},
	{"testhb", []byte("foo"), badHeader(), nil, 3,
		&respError{StatusCode: 400, Message: "The Encryption request you specified is not valid. Supported value: AES256."}},
	{"nomd5", []byte("foo"), goodHeader(),
		&Config{Concurrency: 1, PartSize: 5 * mb, NTry: 1, Md5Check: false, Scheme: "http", Client: http.DefaultClient}, 3, nil},
	{"noconc", []byte("foo"), nil,
		&Config{Concurrency: 0, PartSize: 5 * mb, NTry: 1, Md5Check: true, Scheme: "https", Client: ClientWithTimeout(5 * time.Second)}, 3, nil},
}

func TestPutWriter(t *testing.T) {
	b, err := testBucket()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range putTests {
		w, err := b.PutWriter(tt.path, tt.header, tt.config)
		if err != nil {
			errComp(tt.err, err, t, tt)
			continue
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
		errComp(tt.err, err, t, tt)
	}
}

type putMultiTest struct {
	path   string
	data   io.Reader
	header http.Header
	config *Config
	wSize  int64
	err    error
}

var putMultiTests = []putMultiTest{
	{"5mb_test.test", &randSrc{Size: int(5 * mb)}, goodHeader(), nil, 5 * mb, nil},
	{"20mb_test.test", &randSrc{Size: int(20 * mb)}, goodHeader(),
		&Config{Concurrency: 1, PartSize: 5 * mb, NTry: 2, Md5Check: true, Scheme: "https",
			Client: ClientWithTimeout(5 * time.Second)}, 20 * mb, nil},
	{"timeout.test", &randSrc{Size: int(5 * mb)}, goodHeader(),
		&Config{Concurrency: 1, PartSize: 5 * mb, NTry: 1, Md5Check: false, Scheme: "https",
			Client: ClientWithTimeout(1 * time.Millisecond)}, 5 * mb,
		errors.New("timeout")},
	{"timeout.test", &randSrc{Size: int(10 * mb)}, goodHeader(),
		&Config{Concurrency: 1, PartSize: 5 * mb, NTry: 1, Md5Check: true, Scheme: "https",
			Client: ClientWithTimeout(100 * time.Millisecond)}, 10 * mb,
		errors.New("timeout")},
	{"smallpart", &randSrc{Size: int(10 * mb)}, goodHeader(),
		&Config{Concurrency: 4, PartSize: 4 * mb, NTry: 3, Md5Check: false, Scheme: "https",
			Client: ClientWithTimeout(5 * time.Second)}, 10 * mb, nil},
}

func TestPutMulti(t *testing.T) {
	t.Parallel()
	b, err := testBucket()
	if err != nil {
		t.Fatal(err)
	}
	SetLogger(os.Stderr, "test: ", (log.LstdFlags | log.Lshortfile), true)
	for _, tt := range putMultiTests {
		w, err := b.PutWriter(tt.path, tt.header, tt.config)
		if err != nil {
			errComp(tt.err, err, t, tt)
			continue
		}
		n, err := io.Copy(w, tt.data)
		if err != nil {
			t.Error(err)
		}
		if n != tt.wSize {
			t.Errorf("Expected size: %d. Actual: %d", tt.wSize, n)

		}
		err = w.Close()
		errComp(tt.err, err, t, tt)
	}
}

type tB struct {
	*Bucket
}

func testBucket() (*tB, error) {
	k, err := InstanceKeys()
	if err != nil {
		k, err = EnvKeys()
		if err != nil {
			return nil, err
		}
	}
	bucket := os.Getenv("TEST_BUCKET")
	if bucket == "" {
		return nil, errors.New("TEST_BUCKET must be set in environment")

	}
	s3 := New("", k)
	b := tB{s3.Bucket(bucket)}

	return &b, err
}

func (b *tB) putReader(path string, r *io.Reader) error {

	if r == nil {
		return nil // special handling for nil case
	}

	w, err := b.PutWriter(path, nil, nil)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, *r)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return nil
}

func errComp(expect, actual error, t *testing.T, tt interface{}) bool {

	if expect == nil && actual == nil {
		return true
	}

	if expect == nil || actual == nil {
		t.Errorf("PutWriter called with %v\n Expected: %v\n Actual:   %v\n", tt, expect, actual)
		return false
	}
	if !strings.Contains(actual.Error(), expect.Error()) {
		t.Errorf("PutWriter called with %v\n Expected: %v\n Actual:   %v\n", tt, expect, actual)
		return false
	}
	return true

}

func goodHeader() http.Header {
	header := make(http.Header)
	header.Add("x-amz-server-side-encryption", "AES256")
	header.Add("x-amz-meta-foometadata", "testmeta")
	return header
}

func badHeader() http.Header {
	header := make(http.Header)
	header.Add("x-amz-server-side-encryption", "AES512")
	return header
}

type randSrc struct {
	Size  int
	total int
}

func (r *randSrc) Read(p []byte) (int, error) {

	n, err := rand.Read(p)
	r.total = r.total + n
	if r.total >= r.Size {
		return n - (r.total - r.Size), io.EOF
	}
	return n, err
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
