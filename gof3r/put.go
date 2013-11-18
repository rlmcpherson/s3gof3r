package main

import (
	"github.com/rlmcpherson/s3gof3r"
	"io"
	"log"
	"net/http"
	"os"
)

type Put struct {
	Path string `short:"f" long:"path" description:"Path to file. Defaults to stdin for streaming." default:"/dev/stdin"`
	CommonOpts
}

var put Put

func (put *Put) Execute(args []string) error {
	conf := new(s3gof3r.Config)
	conf = s3gof3r.DefaultConfig
	k := s3gof3r.Keys{AccessKey: os.Getenv("AWS_ACCESS_KEY"),
		SecretKey: os.Getenv("AWS_SECRET_KEY"),
	}
	s3 := s3gof3r.New(s3gof3r.DefaultDomain, k)
	b := s3.Bucket(put.Bucket)
	if put.Concurrency > 0 {
		conf.Concurrency = put.Concurrency
	}
	conf.PartSize = get.PartSize
	conf.Md5Check = !put.CheckDisable
	log.Println(put)
	if put.Header == nil {
		put.Header = make(http.Header)
	}

	w, err := b.PutWriter(put.Key, put.Header, conf)
	if err != nil {
		return err
	}
	r, err := os.Open(put.Path)
	if err != nil {
		return err
	}
	defer r.Close()
	if _, err = io.Copy(w, r); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	if put.Debug {
		debug()
	}
	return nil
}

func init() {
	put.Path = "/dev/stdin" // TODO: figure out how to use defaults in struct
	put.Concurrency = s3gof3r.DefaultConfig.Concurrency
	put.PartSize = s3gof3r.DefaultConfig.PartSize
	parser = parser.AddCommand("put", "put (upload) to S3", "put (upload)to S3", &put)
}
