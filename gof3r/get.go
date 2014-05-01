package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	"github.com/rlmcpherson/s3gof3r"
)

type Get struct {
	Path string `short:"p" long:"path" description:"Path to file. Defaults to standard output for streaming."`
	CommonOpts
	VersionId string `short:"v" long:"versionId" description:"The version ID of the object. Not compatible with md5 checking."`
}

var get Get

func (get *Get) Execute(args []string) (err error) {
	conf := new(s3gof3r.Config)
	*conf = *s3gof3r.DefaultConfig
	k, err := getAWSKeys()
	if err != nil {
		return
	}
	s3 := s3gof3r.New(get.EndPoint, k)
	b := s3.Bucket(get.Bucket)
	if get.Concurrency > 0 {
		conf.Concurrency = get.Concurrency
	}
	conf.PartSize = get.PartSize
	conf.Md5Check = !get.CheckDisable
	get.Key = url.QueryEscape(get.Key)

	s3gof3r.SetLogger(os.Stderr, "", log.LstdFlags, get.Debug)

	if get.VersionId != "" {
		get.Key = fmt.Sprintf("%s?versionId=%s", get.Key, get.VersionId)
	}

	w, err := os.Create(get.Path)
	if err != nil {
		if get.Path == "" {
			w = os.Stdout
		} else {
			return
		}
	}
	defer w.Close()
	r, header, err := b.GetReader(get.Key, conf)
	if err != nil {
		return
	}
	if _, err = io.Copy(w, r); err != nil {
		return
	}
	if err = r.Close(); err != nil {
		return
	}
	if get.Debug {
		log.Println("Headers: ", header)
		debug()
	}
	return
}

func init() {
	_, err := parser.AddCommand("get", "get (download) from S3", "get (download) from S3", &get)
	if err != nil {
		log.Fatal(err)
	}
}
