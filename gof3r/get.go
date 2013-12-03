package main

import (
	"github.com/rlmcpherson/s3gof3r"
	"io"
	"log"
	"os"
)

type Get struct {
	Path string `short:"p" long:"path" description:"Path to file. Defaults to standard output for streaming." default:"/dev/stdout"`
	CommonOpts
}

var get Get

func (get *Get) Execute(args []string) (err error) {
	conf := new(s3gof3r.Config)
	conf = s3gof3r.DefaultConfig
	k := getKeys()
	s3 := s3gof3r.New(s3gof3r.DefaultDomain, k)
	b := s3.Bucket(get.Bucket)
	if get.Concurrency > 0 {
		conf.Concurrency = get.Concurrency
	}
	conf.PartSize = get.PartSize
	conf.Md5Check = !get.CheckDisable
	log.Println("GET: ", get)
	r, header, err := b.GetReader(get.Key, conf)
	if err != nil {
		return
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
	if _, err = io.Copy(w, r); err != nil {
		return
	}
	if err = r.Close(); err != nil {
		return
	}
	log.Println("Headers: ", header)
	if get.Debug {
		debug()
	}
	return
}

func init() {
	// TODO: figure out how to use defaults in struct
	get.Concurrency = s3gof3r.DefaultConfig.Concurrency
	get.PartSize = s3gof3r.DefaultConfig.PartSize
	_, err := parser.AddCommand("get", "get (download) from S3", "get (download) from S3", &get)
	if err != nil {
		log.Fatal(err)
	}
}
