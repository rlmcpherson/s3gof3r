package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/rlmcpherson/s3gof3r"
)

type Put struct {
	Key    string `long:"key" short:"k" description:"S3 object key" required:"true"`
	Bucket string `long:"bucket" short:"b" description:"S3 bucket" required:"true"`
	Path   string `short:"p" long:"path" description:"Path to file. Defaults to standard input for streaming."`
	CommonOpts
	Header http.Header `long:"header" short:"m" description:"HTTP headers. May be used to set custom metadata, server-side encryption etc."`
}

var put Put

func (put *Put) Execute(args []string) (err error) {
	conf := new(s3gof3r.Config)
	*conf = *s3gof3r.DefaultConfig
	k, err := getAWSKeys()
	if err != nil {
		return
	}
	s3 := s3gof3r.New(put.EndPoint, k)
	b := s3.Bucket(put.Bucket)
	conf.Concurrency = put.Concurrency
	if put.NoSSL {
		conf.Scheme = "http"
	}
	conf.PartSize = put.PartSize
	conf.Md5Check = !put.NoMd5
	s3gof3r.SetLogger(os.Stderr, "", log.LstdFlags, put.Debug)

	if put.Header == nil {
		put.Header = make(http.Header)
	}

	r, err := os.Open(put.Path)
	if err != nil {
		if put.Path == "" {
			r = os.Stdin
		} else {
			return
		}
	}
	defer r.Close()
	w, err := b.PutWriter(put.Key, put.Header, conf)
	if err != nil {
		return
	}
	if _, err = io.Copy(w, r); err != nil {
		return
	}
	if err = w.Close(); err != nil {
		return
	}
	return
}

func init() {
	_, err := parser.AddCommand("put", "upload to S3", "put (upload) data to S3 object", &put)
	if err != nil {
		log.Fatal(err)
	}
}
