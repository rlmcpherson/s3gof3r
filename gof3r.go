package s3gof3r

import (
	//"io"
	"net/http"
	//"os"
)

type Keys struct {
	AccessKey string
	SecretKey string
}

type S3 struct {
	// The service's domain. Defaults to "amazonaws.com"
	Domain string
	Keys
}

func New(domain string, keys Keys) {
}

type Bucket struct {
	*S3
	Name string
}

type Config struct {
	*http.Client
	Concurrency int
	NTry        int
	Md5         Md5Check
}

// Defaults
var DefaultConfig = &Config{
	Concurrency: 20,
	NTry:        5,
	Md5:         File,
}

var DefaultDomain = "amazonaws.com"

type Md5Check uint

const (
	File Md5Check = iota
	Metadata
	None
)

func (*Bucket) GetReader() {}

func (*Bucket) PutWriter() {}

func (*S3) sign() {}
