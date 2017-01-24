package main

import (
	"log"

	"github.com/rlmcpherson/s3gof3r"
)

var info infoOpts

type infoOpts struct {
    CommonOpts
	Key       string `long:"key" short:"k" description:"S3 object key" required:"true" no-ini:"true"`
	Bucket    string `long:"bucket" short:"b" description:"S3 bucket" required:"true" no-ini:"true"`
	VersionID string `short:"v" long:"versionId" description:"Version ID of the object. Incompatible with md5 check (use --no-md5)." no-ini:"true"`
}

func (info *infoOpts) Execute(args []string) (err error) {
	conf := new(s3gof3r.Config)
	*conf = *s3gof3r.DefaultConfig
	k, err := getAWSKeys()
	if err != nil {
		return err
	}

	s3 := s3gof3r.New(info.EndPoint, k)
	b := s3.Bucket(info.Bucket)

	resp, err := s3gof3r.GetInfo(b, s3gof3r.DefaultConfig, info.Key, info.VersionID)

	if err != nil {
		return err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("Found object:")
		log.Println("  Status: ", resp.StatusCode)
		log.Println("  Last-Modified: ", resp.Header["Last-Modified"])
		log.Println("  Size: ", resp.Header["Content-Length"])
	} else if resp.StatusCode == 403 {
		log.Fatal("Access Denied")
	} else {
		log.Fatal("Non-2XX status code: ", resp.StatusCode)
	}

	return nil
}

func init() {
	_, err := parser.AddCommand("info", "check info from S3", "get information about an object from S3", &info)

	if err != nil {
		log.Fatal(err)
	}
}
