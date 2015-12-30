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
	"sync"
	"syscall"
	"testing"
	"time"
)

var b *tB

func init() {
	var err error
	b, err = testBucket()
	if err != nil {
		log.Fatal(err)
	}
	uploadTestFiles()
	if testing.Verbose() {
		SetLogger(os.Stderr, "test: ", (log.LstdFlags | log.Lshortfile), true)
	}
}

func uploadTestFiles() {
	var wg sync.WaitGroup
	for _, tt := range getTests {
		if tt.rSize >= 0 {
			wg.Add(1)
			go func(path string, rSize int64) {
				err := b.putReader(path, &randSrc{Size: int(rSize)})
				if err != nil {
					log.Fatal(err)
				}
				wg.Done()
			}(tt.path, tt.rSize)
		}
	}
	wg.Wait()
}

var getTests = []struct {
	path   string
	config *Config
	rSize  int64
	err    error
}{
	{"t1.test", nil, 1 * kb, nil},
	{"no-md5", &Config{Scheme: "https", Client: ClientWithTimeout(clientTimeout), Md5Check: false}, 1, nil},
	{"NoKey", nil, -1, &RespError{StatusCode: 404, Message: "The specified key does not exist."}},
	{"", nil, -1, fmt.Errorf("empty path requested")},
	{"1_mb_test",
		&Config{Concurrency: 2, PartSize: 5 * mb, NTry: 2, Md5Check: true, Scheme: "https", Client: ClientWithTimeout(2 * time.Second)},
		1 * mb,
		nil},
	{"b1", nil, 1, nil},
	{"0byte", &Config{Scheme: "https", Client: ClientWithTimeout(clientTimeout), Md5Check: false}, 0, nil},
}

func TestGetReader(t *testing.T) {
	t.Parallel()

	for _, tt := range getTests {
		r, h, err := b.GetReader(tt.path, tt.config)
		if err != nil {
			errComp(tt.err, err, t, tt)
			continue
		}
		t.Logf("headers for %s: %v\n", tt.path, h)
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

func TestPutWriter(t *testing.T) {
	t.Parallel()
	var putTests = []struct {
		path   string
		data   []byte
		header http.Header
		config *Config
		wSize  int64
		err    error
	}{
		{"testfile", []byte("test_data"), nil, nil, 9, nil},
		{"", []byte("test_data"), nil, nil,
			9, &RespError{StatusCode: 400, Message: "A key must be specified"}},
		{"test0byte", []byte(""), nil, nil, 0, nil},
		{"testhg", []byte("foo"), goodHeader(), nil, 3, nil},
		{"testhb", []byte("foo"), badHeader(), nil, 3,
			&RespError{StatusCode: 400, Message: "The encryption method specified is not supported"}},
		{"nomd5", []byte("foo"), goodHeader(),
			&Config{Concurrency: 1, PartSize: 5 * mb, NTry: 1, Md5Check: false, Scheme: "http", Client: http.DefaultClient}, 3, nil},
		{"noconc", []byte("foo"), nil,
			&Config{Concurrency: 0, PartSize: 5 * mb, NTry: 1, Md5Check: true, Scheme: "https", Client: ClientWithTimeout(5 * time.Second)}, 3, nil},
		{"enc test", []byte("test_data"), nil, nil, 9, nil},
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

type multiTest struct {
	path   string
	data   io.Reader
	header http.Header
	config *Config
	wSize  int64
	err    error
}

// Tests of multipart puts and gets
// Since the minimum part size is 5 mb, these take longer to run
// These tests can be skipped by running test with the short flag
func TestMulti(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping, short mode")
	}

	t.Parallel()
	var putMultiTests = []multiTest{
		{"1mb_test.test", &randSrc{Size: int(1 * mb)}, goodHeader(), nil, 1 * mb, nil},
		{"21mb_test.test", &randSrc{Size: int(21 * mb)}, goodHeader(),
			&Config{Concurrency: 3, PartSize: 5 * mb, NTry: 2, Md5Check: true, Scheme: "https",
				Client: ClientWithTimeout(5 * time.Second)}, 21 * mb, nil},
		{"timeout.test1", &randSrc{Size: int(1 * mb)}, goodHeader(),
			&Config{Concurrency: 1, PartSize: 5 * mb, NTry: 1, Md5Check: false, Scheme: "https",
				Client: ClientWithTimeout(1 * time.Millisecond)}, 1 * mb,
			errors.New("timeout")},
		{"timeout.test2", &randSrc{Size: int(1 * mb)}, goodHeader(),
			&Config{Concurrency: 1, PartSize: 5 * mb, NTry: 1, Md5Check: true, Scheme: "https",
				Client: ClientWithTimeout(1 * time.Millisecond)}, 1 * mb,
			errors.New("timeout")},
		{"toosmallpart", &randSrc{Size: int(6 * mb)}, goodHeader(),
			&Config{Concurrency: 4, PartSize: 5 * mb, NTry: 3, Md5Check: false, Scheme: "https",
				Client: ClientWithTimeout(2 * time.Second)}, 6 * mb, nil},
	}
	var wg sync.WaitGroup
	for _, tt := range putMultiTests {
		w, err := b.PutWriter(tt.path, tt.header, tt.config)
		if err != nil {
			errComp(tt.err, err, t, tt)
			continue
		}
		wg.Add(1)

		go func(w io.WriteCloser, tt multiTest) {
			n, err := io.Copy(w, tt.data)
			if err != nil {
				t.Error(err)
			}
			if n != tt.wSize {
				t.Errorf("Expected size: %d. Actual: %d", tt.wSize, n)

			}
			err = w.Close()
			errComp(tt.err, err, t, tt)
			r, h, err := b.GetReader(tt.path, tt.config)
			if err != nil {
				errComp(tt.err, err, t, tt)
				//return
			}
			t.Logf("headers %v\n", h)
			gw := ioutil.Discard

			n, err = io.Copy(gw, r)
			if err != nil {
				t.Error(err)
			}
			if n != tt.wSize {
				t.Errorf("Expected size: %d. Actual: %d", tt.wSize, n)

			}
			t.Logf("got %s", tt.path)
			err = r.Close()
			errComp(tt.err, err, t, tt)
			wg.Done()
		}(w, tt)
	}
	wg.Wait()
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

func (b *tB) putReader(path string, r io.Reader) error {
	if r == nil {
		return nil // special handling for nil case
	}

	w, err := b.PutWriter(path, nil, nil)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
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
		t.Errorf("called with %v\n Expected: %v\n Actual:   %v\n", tt, expect, actual)
		return false
	}
	if !strings.Contains(actual.Error(), expect.Error()) {
		t.Errorf("called with %v\n Expected: %v\n Actual:   %v\n", tt, expect, actual)
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

func TestDelete(t *testing.T) {

	var deleteTests = []struct {
		path  string
		exist bool
		err   error
	}{
		{"delete1", true, nil},
		{"delete 2", false, nil},
		{"/delete 2", false, nil},
	}

	for _, tt := range deleteTests {
		if tt.exist {
			err := b.putReader(tt.path, &randSrc{Size: 1})

			if err != nil {
				t.Fatal(err)
			}
		}
		err := b.Delete(tt.path)
		t.Log(err)
		errComp(tt.err, err, t, tt)
	}
}

func TestGetVersion(t *testing.T) {
	t.Parallel()

	var versionTests = []struct {
		path string
		err  error
	}{
		{"key1", nil},
	}
	for _, tt := range versionTests {
		if err := b.putReader(tt.path, &randSrc{Size: 1}); err != nil {
			t.Fatal(err)
		}
		// get version id
		r, h, err := b.GetReader(tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		r.Close()
		v := h.Get("x-amz-version-id")
		if v == "" {
			t.Logf("versioning not enabled on %s\n", b.Name)
			t.SkipNow()
		}
		// upload again for > 1 version
		if err := b.putReader(tt.path, &randSrc{Size: 1}); err != nil {
			t.Fatal(err)
		}

		// request first uploaded version
		t.Logf("version id: %s", v)
		p := fmt.Sprintf("%s?versionId=%s", tt.path, v)
		r, _, err = b.GetReader(p, nil)
		if err != nil {
			t.Fatal(err)
		}
		r.Close()
		errComp(tt.err, err, t, tt)
	}
}

func TestPutWriteAfterClose(t *testing.T) {
	t.Parallel()

	w, err := b.PutWriter("test", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Write(make([]byte, 10))
	if err != syscall.EINVAL {
		t.Errorf("expected %v on write after close, got %v", syscall.EINVAL, err)
	}
}

func TestGetReadAfterClose(t *testing.T) {
	t.Parallel()

	r, _, err := b.GetReader("test", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = r.Close()
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Read(make([]byte, 10))
	if err != syscall.EINVAL {
		t.Errorf("expected %v on read after close, got %v", syscall.EINVAL, err)
	}
}

// Test Close when downloading of parts still in progress
func TestGetCloseBeforeRead(t *testing.T) {
	r, _, err := b.GetReader(getTests[4].path, getTests[4].config)
	if err != nil {
		t.Fatal(err)
	}
	//terr := fmt.Errorf("read error: 0 bytes read. expected: %d", getTests[4].rSize)
	terr := fmt.Errorf("read error: %d bytes read. expected: %d", 0, getTests[4].rSize)
	tmr := time.NewTimer(100 * time.Millisecond)
	defer tmr.Stop()
	closed := make(chan struct{})
	go func() {
		err = r.Close()
		close(closed)
		if err != nil && err.Error() != terr.Error() || err == nil {
			t.Errorf("expected error %v on Close, got %v", terr, err)
		}
	}()

	// fail test if close does not return before timeout
	select {
	case <-closed:
		tmr.Stop()
	case <-tmr.C:
		t.Fatal("getter close did not return before timeout")
	}
}

func TestPutterAfterError(t *testing.T) {
	w, err := b.PutWriter("test", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := w.(*putter)
	if !ok {
		t.Fatal("putter type cast failed")
	}
	terr := fmt.Errorf("test error")
	p.err = terr
	_, err = w.Write([]byte("foo"))
	if err != terr {
		t.Errorf("expected error %v on Write, got %v", terr, err)
	}
	err = w.Close()
	if err != terr {
		t.Errorf("expected error %v on Close, got %v", terr, err)
	}
}

func TestGetterAfterError(t *testing.T) {
	r, _, err := b.GetReader("test", nil)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := r.(*getter)
	if !ok {
		t.Fatal("getter type cast failed")
	}
	terr := fmt.Errorf("test error")
	g.err = terr
	_, err = r.Read([]byte("foo"))
	if err != terr {
		t.Errorf("expected error %v on Read, got %v", terr, err)
	}
	err = r.Close()
	if err != terr {
		t.Errorf("expected error %v on Close, got %v", terr, err)
	}
}

var regionsTests = []struct {
	domain string
	region string
	err    error
}{
	{domain: "s3.amazonaws.com", region: "us-east-1"},
	{domain: "s3-external-1.amazonaws.com", region: "us-east-1"},
	{domain: "s3-sa-east-1.amazonaws.com", region: "sa-east-1"},
}

func TestRegion(t *testing.T) {
	for _, tt := range regionsTests {
		s3 := &S3{Domain: tt.domain}
		region := s3.Region()
		if region != tt.region {
			t.Errorf("wrong region detected, got '%s', expected '%s'", region, tt.region)
		}
	}
}

func TestBucketURL(t *testing.T) {
	var urlTests = []struct {
		bucket string
		path   string
		config *Config
		url    string
	}{
		{"bucket1", "path", DefaultConfig, "https://bucket1.s3.amazonaws.com/path"},
		{"bucket1", "#path", DefaultConfig, `https://bucket1.s3.amazonaws.com/%23path`},
		{"bucket1", "#path ", DefaultConfig, `https://bucket1.s3.amazonaws.com/%23path%20`},
		{"bucket.2", "path", DefaultConfig, "https://s3.amazonaws.com/bucket.2/path"},
		{"bucket.2", "#path", DefaultConfig, `https://s3.amazonaws.com/bucket.2/%23path`},
		{"bucket.2", "#path?versionId=seQK1YwRAy6Ex25YHb_yJHbo94jSDnpu", DefaultConfig, `https://s3.amazonaws.com/bucket.2/%23path%3FversionId=seQK1YwRAy6Ex25YHb_yJHbo94jSDnpu`}, // versionId-specific handling
	}

	for _, tt := range urlTests {
		s3 := New("", Keys{})
		b := s3.Bucket(tt.bucket)
		u, err := b.url(tt.path, tt.config)
		if err != nil {
			t.Error(err)
		}
		if u.String() != tt.url {
			t.Errorf("got '%s', expected '%s'", u.String(), tt.url)
		}

	}

}

// reduce parallelism and part size to benchmark
// memory pool reuse
func benchConfig() *Config {
	var conf Config
	conf = *DefaultConfig
	conf.Concurrency = 4
	conf.PartSize = 5 * mb
	return &conf
}

func BenchmarkPut(k *testing.B) {
	r := &randSrc{Size: int(300 * mb)}
	k.ReportAllocs()
	for i := 0; i < k.N; i++ {
		w, _ := b.PutWriter("bench_test", nil, benchConfig())
		n, err := io.Copy(w, r)
		if err != nil {
			k.Fatal(err)
		}
		k.SetBytes(n)
		w.Close()
	}
}

func BenchmarkGet(k *testing.B) {
	k.ReportAllocs()
	for i := 0; i < k.N; i++ {
		r, _, _ := b.GetReader("bench_test", benchConfig())
		n, err := io.Copy(ioutil.Discard, r)
		if err != nil {
			k.Fatal(err)
		}
		k.SetBytes(n)
		r.Close()
	}
}
