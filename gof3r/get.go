package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/rlmcpherson/s3gof3r"
)

type getOpts struct {
	Key    string `long:"key" short:"k" description:"S3 object key" required:"true" no-ini:"true"`
	Bucket string `long:"bucket" short:"b" description:"S3 bucket" required:"true" no-ini:"true"`
	Path   string `short:"p" long:"path" description:"Path to file. Defaults to standard output for streaming." no-ini:"true"`
	DataOpts
	CommonOpts
	VersionID string `short:"v" long:"versionId" description:"Version ID of the object. Incompatible with md5 check (use --no-md5)." no-ini:"true"`
}

var get getOpts

func (get *getOpts) Execute(args []string) (err error) {
	conf := new(s3gof3r.Config)
	*conf = *s3gof3r.DefaultConfig
	k, err := getAWSKeys()
	if err != nil {
		return
	}
	s3 := s3gof3r.New(get.EndPoint, k)
	b := s3.Bucket(get.Bucket)
	conf.Concurrency = get.Concurrency
	if get.NoSSL {
		conf.Scheme = "http"
	}
	conf.PartSize = get.PartSize
	conf.Md5Check = !get.NoMd5

	s3gof3r.SetLogger(os.Stderr, "", log.LstdFlags, get.Debug)

	if get.VersionID != "" {
		get.Key = fmt.Sprintf("%s?versionId=%s", get.Key, get.VersionID)
	}

	w, err := os.Create(get.Path)
	if err != nil {
		if get.Path == "" {
			w = os.Stdout
		} else {
			return
		}
	}
	defer checkClose(w, err)
	r, header, err := b.GetReader(get.Key, conf)
	if err != nil {
		return
	}
	defer checkClose(r, err)
	if _, err = io.Copy(w, r); err != nil {
		return
	}
	if get.Debug {
		log.Println("Headers: ", header)
	}
	return
}

func init() {
	_, err := parser.AddCommand("get", "download from S3", "get (download) object from S3", &get)
	if err != nil {
		log.Fatal(err)
	}
}
