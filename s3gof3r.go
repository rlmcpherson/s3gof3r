// Package s3gof3r provides fast, parallelized, streaming access to Amazon S3. It includes a command-line interface: `gof3r`.

package s3gof3r

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

// S3 contains the domain or endpoint of an S3-compatible service and
// the authentication keys for that service.
type S3 struct {
	Domain string // The s3-compatible endpoint. Defaults to "s3.amazonaws.com"
	Keys
}

// A Bucket for an S3 service.
type Bucket struct {
	*S3
	Name string
}

// Config includes configuration parameters for s3gof3r
type Config struct {
	*http.Client       // nil to use s3gof3r default client
	Concurrency  int   // number of parts to get or put concurrently
	PartSize     int64 // initial  part size in bytes to use for multipart gets or puts
	NTry         int   // maximum attempts for each part
	Md5Check     bool  // The md5 hash of the object is stored in <bucket>/.md5/<object_key>.md5
	// When true, it is stored on puts and verified on gets
	Scheme string // url scheme, defaults to 'https'
}

// DefaultConfig contains defaults used if *Config is nil
var DefaultConfig = &Config{
	Concurrency: 10,
	PartSize:    20 * mb,
	NTry:        10,
	Md5Check:    true,
	Scheme:      "https",
}

// http client timeout
const (
	clientTimeout = 5 * time.Second
)

// DefaultDomain is set to the endpoint for the U.S. S3 service.
var DefaultDomain = "s3.amazonaws.com"

// New Returns a new S3
// domain defaults to DefaultDomain if empty
func New(domain string, keys Keys) *S3 {
	if domain == "" {
		domain = DefaultDomain
	}
	return &S3{domain, keys}
}

// Bucket returns a bucket on s3
func (s3 *S3) Bucket(name string) *Bucket {
	return &Bucket{s3, name}
}

// GetReader provides a reader and downloads data using parallel ranged get requests.
// Data from the requests are ordered and written sequentially.
//
// Data integrity is verified via the option specified in c.
// Header data from the downloaded object is also returned, useful for reading object metadata.
// DefaultConfig is used if c is nil
func (b *Bucket) GetReader(path string, c *Config) (r io.ReadCloser, h http.Header, err error) {
	if c == nil {
		c = DefaultConfig
	}
	if c.Client == nil {
		c.Client = ClientWithTimeout(clientTimeout)
	}
	return newGetter(b.Url(path, c), c, b)
}

// PutWriter provides a writer to upload data as multipart upload requests.
//
// Each header in h is added to the HTTP request header. This is useful for specifying
// options such as server-side encryption in metadata as well as custom user metadata.
// DefaultConfig is used if c is nil.
func (b *Bucket) PutWriter(path string, h http.Header, c *Config) (w io.WriteCloser, err error) {
	if c == nil {
		c = DefaultConfig
	}
	if c.Client == nil {
		c.Client = ClientWithTimeout(clientTimeout)
	}
	return newPutter(b.Url(path, c), h, c, b)
}

// Url returns a parsed url to the given path, using the scheme specified in Config.Scheme
func (b *Bucket) Url(path string, c *Config) url.URL {
	url_, err := url.Parse(fmt.Sprintf("%s://%s.%s/%s", c.Scheme, b.Name, b.S3.Domain, path))
	if err != nil {
		panic(err)
	}
	return *url_
}

// SetLogger wraps the standard library log package.
//
// It allows the internal logging of s3gof3r to be set to a desired output and format.
// Setting debug to true enables debug logging output. s3gof3r does not log output by default.
func SetLogger(out io.Writer, prefix string, flag int, debug bool) {
	logger = internalLogger{
		log.New(out, prefix, flag),
		debug,
	}
}

type internalLogger struct {
	*log.Logger
	debug bool
}

var logger internalLogger

func (l *internalLogger) debugPrintln(v ...interface{}) {
	if logger.debug {
		logger.Println(v...)
	}
}

func (l *internalLogger) debugPrintf(format string, v ...interface{}) {
	if logger.debug {
		logger.Printf(format, v...)
	}
}

// Initialize internal logger to log to no-op (ioutil.Discard) by default.
func init() {
	logger = internalLogger{
		log.New(ioutil.Discard, "", log.LstdFlags),
		false,
	}
}
