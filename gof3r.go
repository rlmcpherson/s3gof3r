// Package S3 provides fast concurrent access to Amazon S3, including CLI.
package s3gof3r

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Keys for an Amazon Web Services account.
// Used for signing http requests.
type Keys struct {
	AccessKey string
	SecretKey string
}

type S3 struct {
	Domain string // The service's domain. Defaults to "s3.amazonaws.com"
	Keys
}

type Bucket struct {
	*S3
	Name string
}

type Config struct {
	*http.Client       // nil to use s3gof3r default client
	Concurrency  int   // number of parts to get or put concurrently
	PartSize     int64 //  initial  part size in bytes to use for multipart gets or puts
	NTry         int   // maximum attempts for each part
	Md5Check     bool  // the md5 hash of the object is stored in <bucket>/.md5/<object_key> and verified on gets
}

// Defaults
var DefaultConfig = &Config{
	Concurrency: 30,
	PartSize:    20 * mb,
	NTry:        5,
	Md5Check:    true,
}

var DefaultDomain = "s3.amazonaws.com"

// http client timeout settings
const (
	clientDialTimeout     = 2 * time.Second
	responseHeaderTimeout = 5 * time.Second
)

// Returns a new S3
// domain defaults to DefaultDomain if empty
func New(domain string, keys Keys) *S3 {
	if domain == "" {
		domain = DefaultDomain
	}
	return &S3{domain, keys}
}

// Returns a bucket on s3j
func (s3 *S3) Bucket(name string) *Bucket {
	return &Bucket{s3, name}
}

// Provides a reader and downloads data using parallel ranged get requests.
// Data from the requests is reordered and written sequentially.
//
// Data integrity is verified via the option specified in c.
// Header data from the downloaded object is also returned, useful for reading object metadata.
func (b *Bucket) GetReader(path string, c *Config) (r io.ReadCloser, h http.Header, err error) {
	if c == nil {
		c = DefaultConfig
	}
	if c.Client == nil {
		c.Client = createClientWithTimeout(clientDialTimeout)
	}
	var url_ *url.URL
	url_, err = url.Parse(fmt.Sprintf("https://%s.%s%s", b.Name, b.S3.Domain, path))
	if err != nil {
		return
	}
	log.Print("S3: ", b.S3)
	log.Print("Bucket: ", b)
	log.Print("Path: ", path)
	log.Print("URL: ", url_)
	return newGetter(*url_, c, b)
}

// Provides a writer to upload data as multipart upload requests.
//
// Each header in h is added to the HTTP request header. This is useful for specifying
// options such as server-side encryption in metadata as well as custom user metadata.
// DefaultConfig is used if c is nil.
func (b *Bucket) PutWriter(path string, h http.Header, c *Config) (w io.WriteCloser, err error) {
	if c == nil {
		c = DefaultConfig
	}
	if c.Client == nil {
		c.Client = createClientWithTimeout(clientDialTimeout)
	}
	var url_ *url.URL
	url_, err = url.Parse(fmt.Sprintf("https://%s.%s%s", b.Name, b.S3.Domain, path))
	if err != nil {
		return
	}
	log.Println("S3: ", b.S3)
	log.Println("Bucket: ", b)
	log.Println("Path: ", path)
	log.Println("URL: ", url_)
	log.Println("Header: ", h)
	log.Println("Config: ", c)
	return newPutter(*url_, h, c, b)
}

func createClientWithTimeout(timeout time.Duration) *http.Client {
	dialFunc := func(network, addr string) (net.Conn, error) {
		c, err := net.DialTimeout(network, addr, timeout)
		if err != nil {
			log.Print(err) // for debugging
			return nil, err
		}
		return c, nil
	}

	return &http.Client{
		Transport: &http.Transport{
			Dial: dialFunc,
			ResponseHeaderTimeout: responseHeaderTimeout,
			//MaxIdleConnsPerHost:   5,
		},
	}
}
