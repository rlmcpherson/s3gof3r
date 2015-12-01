package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/rlmcpherson/s3gof3r"
)

type CpArg struct {
	Source string `name:"source"`
	Dest   string `name:"dest"`
}

type cpOpts struct {
	DataOpts
	CommonOpts
	CpArg `positional-args:"true" required:"true"`
	UpOpts
}

var cp cpOpts

func (cp *cpOpts) Execute(args []string) (err error) {

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

	src, err := func(src string) (io.ReadCloser, error) {
		if !strings.HasPrefix(strings.ToLower(src), "s3") {
			return os.Open(src)
		}
		u, err := url.ParseRequestURI(src)
		if err != nil {
			return nil, fmt.Errorf("parse error: %s", err)
		}

		r, _, err := s3.Bucket(u.Host).GetReader(u.Path, conf)
		return r, err
	}(cp.Source)
	if err != nil {
		return
	}
	defer checkClose(src, err)

	dst, err := func(dst string) (io.WriteCloser, error) {
		if !strings.HasPrefix(strings.ToLower(dst), "s3") {
			return os.Create(dst)
		}
		u, err := url.ParseRequestURI(dst)
		if err != nil {
			return nil, fmt.Errorf("parse error: %s", err)
		}

		return s3.Bucket(u.Host).PutWriter(u.Path, ACL(cp.Header, cp.ACL), conf)
	}(cp.Dest)
	if err != nil {
		return
	}

	defer checkClose(dst, err)
	_, err = io.Copy(dst, src)
	return
}

func init() {
	cmd, err := parser.AddCommand("cp", "copy S3 objects", "copy S3 objects to or from S3 and local files", &cp)
	if err != nil {
		log.Fatal(err)
	}
	cmd.Aliases = []string{"copy"}
}
