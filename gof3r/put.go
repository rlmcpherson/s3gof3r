package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/rlmcpherson/s3gof3r"
)

type Put struct {
	Path string `short:"p" long:"path" description:"Path to file. Defaults to standard input for streaming."`
	CommonOpts
	Header http.Header `long:"header" short:"m" description:"HTTP headers"`
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
	if put.Concurrency > 0 {
		conf.Concurrency = put.Concurrency
	}
	conf.PartSize = put.PartSize
	conf.Md5Check = !put.CheckDisable
	put.Key = url.QueryEscape(put.Key)
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
	if put.Debug {
		debug()
	}
	return
}

func init() {
	_, err := parser.AddCommand("put", "put (upload) to S3", "put (upload)to S3", &put)
	if err != nil {
		log.Fatal(err)
	}
}
