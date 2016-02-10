// Package s3gof3r provides fast, parallelized, streaming access to Amazon S3. It includes a command-line interface: `gof3r`.
package s3gof3r

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

const versionParam = "versionId"

var regionMatcher = regexp.MustCompile("s3-([a-z0-9-]+).amazonaws.com")

// S3 contains the domain or endpoint of an S3-compatible service and
// the authentication keys for that service.
type S3 struct {
	Domain string // The s3-compatible endpoint. Defaults to "s3.amazonaws.com"
	Keys
}

// Region returns the service region infering it from S3 domain.
func (s *S3) Region() string {
	switch s.Domain {
	case "s3.amazonaws.com", "s3-external-1.amazonaws.com":
		return "us-east-1"
	default:
		regions := regionMatcher.FindStringSubmatch(s.Domain)
		if len(regions) < 2 {
			if region := os.Getenv("AWS_REGION"); region != "" {
				return region
			}
			panic("can't find endpoint region")
		}
		return regions[1]
	}
}

// A Bucket for an S3 service.
type Bucket struct {
	*S3
	Name string
	*Config
}

// Config includes configuration parameters for s3gof3r
type Config struct {
	*http.Client       // http client to use for requests
	Concurrency  int   // number of parts to get or put concurrently
	PartSize     int64 // initial  part size in bytes to use for multipart gets or puts
	NTry         int   // maximum attempts for each part
	Md5Check     bool  // The md5 hash of the object is stored in <bucket>/.md5/<object_key>.md5
	// When true, it is stored on puts and verified on gets
	Scheme    string // url scheme, defaults to 'https'
	PathStyle bool   // use path style bucket addressing instead of virtual host style
}

// DefaultConfig contains defaults used if *Config is nil
var DefaultConfig = &Config{
	Concurrency: 10,
	PartSize:    20 * mb,
	NTry:        10,
	Md5Check:    true,
	Scheme:      "https",
	Client:      ClientWithTimeout(clientTimeout),
}

// http client timeout
const clientTimeout = 5 * time.Second

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
// Bucket Config is initialized to DefaultConfig
func (s *S3) Bucket(name string) *Bucket {
	return &Bucket{
		S3:     s,
		Name:   name,
		Config: DefaultConfig,
	}
}

// GetReader provides a reader and downloads data using parallel ranged get requests.
// Data from the requests are ordered and written sequentially.
//
// Data integrity is verified via the option specified in c.
// Header data from the downloaded object is also returned, useful for reading object metadata.
// DefaultConfig is used if c is nil
// Callers should call Close on r to ensure that all resources are released.
//
// To specify an object version in a versioned bucket, the version ID may be included in the path as a url parameter. See http://docs.aws.amazon.com/AmazonS3/latest/dev/RetrievingObjectVersions.html
func (b *Bucket) GetReader(path string, c *Config) (r io.ReadCloser, h http.Header, err error) {
	if path == "" {
		return nil, nil, errors.New("empty path requested")
	}
	if c == nil {
		c = b.conf()
	}
	u, err := b.url(path, c)
	if err != nil {
		return nil, nil, err
	}
	return newGetter(*u, c, b)
}

// PutWriter provides a writer to upload data as multipart upload requests.
//
// Each header in h is added to the HTTP request header. This is useful for specifying
// options such as server-side encryption in metadata as well as custom user metadata.
// DefaultConfig is used if c is nil.
// Callers should call Close on w to ensure that all resources are released.
func (b *Bucket) PutWriter(path string, h http.Header, c *Config) (w io.WriteCloser, err error) {
	if c == nil {
		c = b.conf()
	}
	u, err := b.url(path, c)
	if err != nil {
		return nil, err
	}

	return newPutter(*u, h, c, b)
}

// url returns a parsed url to the given path. c must not be nil
func (b *Bucket) url(bPath string, c *Config) (*url.URL, error) {

	// parse versionID parameter from path, if included
	// See https://github.com/rlmcpherson/s3gof3r/issues/84 for rationale
	purl, err := url.Parse(bPath)
	if err != nil {
		return nil, err
	}
	var vals url.Values
	if v := purl.Query().Get(versionParam); v != "" {
		vals = make(url.Values)
		vals.Add(versionParam, v)
		bPath = strings.Split(bPath, "?")[0] // remove versionID from path
	}

	// handling for bucket names containing periods / explicit PathStyle addressing
	// http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html for details
	if strings.Contains(b.Name, ".") || c.PathStyle {
		return &url.URL{
			Host:     b.S3.Domain,
			Scheme:   c.Scheme,
			Path:     path.Clean(fmt.Sprintf("/%s/%s", b.Name, bPath)),
			RawQuery: vals.Encode(),
		}, nil
	} else {
		return &url.URL{
			Scheme:   c.Scheme,
			Path:     path.Clean(fmt.Sprintf("/%s", bPath)),
			Host:     path.Clean(fmt.Sprintf("%s.%s", b.Name, b.S3.Domain)),
			RawQuery: vals.Encode(),
		}, nil
	}
}

func (b *Bucket) conf() *Config {
	c := b.Config
	if c == nil {
		c = DefaultConfig
	}
	return c
}

// Delete deletes the key at path
// If the path does not exist, Delete returns nil (no error).
func (b *Bucket) Delete(path string) error {
	if err := b.delete(path); err != nil {
		return err
	}
	// try to delete md5 file
	if err := b.delete(fmt.Sprintf("/.md5/%s.md5", path)); err != nil {
		return err
	}

	logger.Printf("%s deleted from %s\n", path, b.Name)
	return nil
}

func (b *Bucket) delete(path string) error {
	u, err := b.url(path, b.conf())
	if err != nil {
		return err
	}
	r := http.Request{
		Method: "DELETE",
		URL:    u,
	}
	b.Sign(&r)
	resp, err := b.conf().Do(&r)
	if err != nil {
		return err
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 204 {
		return newRespError(resp)
	}
	return nil
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
