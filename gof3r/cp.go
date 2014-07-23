package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/rlmcpherson/s3gof3r"
)

type Cp struct {
	CommonOpts
	Header http.Header `long:"header" short:"m" description:"HTTP headers. May be used to set custom metadata, server-side encryption etc."`
}

var cp Cp

func (cp *Cp) Usage() string {
	return "<source> <dest> [cp-OPTIONS]"
}

func (cp *Cp) Execute(args []string) (err error) {

	k, err := getAWSKeys()
	if err != nil {
		return
	}

	conf := new(s3gof3r.Config)
	*conf = *s3gof3r.DefaultConfig
	s3 := s3gof3r.New(cp.EndPoint, k)
	conf.Concurrency = cp.Concurrency
	if cp.NoSSL {
		conf.Scheme = "http"
	}
	conf.PartSize = cp.PartSize
	conf.Md5Check = !cp.NoMd5
	s3gof3r.SetLogger(os.Stderr, "", log.LstdFlags, cp.Debug)

	// parse positional cp args
	if len(args) != 2 {
		return fmt.Errorf("cp: source and destination arguments required")
	}

	var urls [2]*url.URL
	for i, a := range args {
		urls[i], err = url.Parse(a)
		if err != nil {
			return fmt.Errorf("parse error: %s", err)
		}
		if urls[i].Host != "" && urls[i].Scheme != "s3" {
			return fmt.Errorf("parse error: %s", urls[i].String())
		}
	}

	src, err := func(src *url.URL) (io.ReadCloser, error) {
		if src.Host == "" {
			return os.Open(src.Path)
		} else {
			r, _, err := s3.Bucket(src.Host).GetReader(src.Path, conf)
			return r, err
		}
	}(urls[0])
	if err != nil {
		return
	}

	dst, err := func(dst *url.URL) (io.WriteCloser, error) {
		if dst.Host == "" {
			return os.Create(dst.Path)
		} else {
			return s3.Bucket(dst.Host).PutWriter(dst.Path, cp.Header, conf)
		}
	}(urls[1])
	if err != nil {
		return
	}

	if _, err = io.Copy(dst, src); err != nil {
		return
	}
	if err = src.Close(); err != nil {
		return
	}
	if err = dst.Close(); err != nil {
		return
	}
	return
}

func init() {
	cmd, err := parser.AddCommand("cp", "copy S3 objects", "copy S3 objects to or from S3 and local files", &cp)
	if err != nil {
		log.Fatal(err)
	}
	cmd.ArgsRequired = true
	cmd.Aliases = []string{"copy"}
}
