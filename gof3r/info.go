package main

import (
	"log"
	"encoding/xml"
	"io/ioutil"
	"github.com/rlmcpherson/s3gof3r"
	"os"
	"strings"
	"fmt"
)

type ListBucketResult struct {
	XMLName  xml.Name   `xml:"ListBucketResult"`
	Contents []Content `xml:"Contents"`
	KeyCount int `xml:KeyCount`
}

type Content struct {
	Name 			string `xml:"Key"`
	ETag 			string `xml:"ETag"`
	LastModified 	string `xml:"LastModified"`
	Size 			string `xml:"Size"`
}

var info infoOpts

type infoOpts struct {
    CommonOpts
	DataOpts
	Key       		string `long:"key" short:"k" description:"S3 object key" required:"true" no-ini:"true"`
	Bucket    		string `long:"bucket" short:"b" description:"S3 bucket" required:"true" no-ini:"true"`
	VersionID 		string `short:"v" long:"versionId" description:"Version ID of the object. Incompatible with md5 check (use --no-md5)." no-ini:"true"`
	ObjectProperty  string `long:"object-property" short:"q" description:"The property requested. Valid values are: etag, last-modified, size" no-ini:"true"`
}

func (info *infoOpts) Execute(args []string) (err error) {
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

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var lbr ListBucketResult
	if err := xml.Unmarshal(respBody, &lbr); err != nil {
		log.Fatal(err)
	}

	// the implementation assumes that the <Key> of the first <Contents> in the response XML
	// should match the requested Key exactly - based on the alphabetically (lexicographic)
	// ascending ordering of the keys in the response from the AWS API
	// otherwise it should fail, as the intent of this method is to obtain info on an exact
	// file of the bucket, even though the method works based on a prefix approach

	// if the intent is to improve on the assumption or if the AWS API stops responding with
	// the keys in alphabetically ascending order and if the ListBucket-access-based approach
	// is to be kept for the `info` command, then the approach should be refactored as to
	// obtain all results given the requested prefix (ask for all pages of data), iterate over
	// them until the requested key is detected - or failing otherwise

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if lbr.KeyCount < 1 {
			os.Exit(1)
		}
		if info.ObjectProperty == "etag" {
			fmt.Println(strings.Replace(lbr.Contents[0].ETag, "\"", "", -1))
		} else if info.ObjectProperty == "last-modified" {
			fmt.Println(lbr.Contents[0].LastModified)
		} else if info.ObjectProperty == "size" {
			fmt.Println(lbr.Contents[0].Size)
		}
	} else if resp.StatusCode == 403 {
		log.Fatal("Access Denied")
		os.Exit(1)
	} else {
		log.Fatal("Non-2XX status code: ", resp.StatusCode)
		os.Exit(1)
	}

	return nil
}

func init() {
	_, err := parser.AddCommand("info", "check info from S3", "get information about an object from S3", &info)

	if err != nil {
		log.Fatal(err)
	}
}
