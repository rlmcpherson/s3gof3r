// Command gof3r provides a command-line interface to Amazon AWS S3.
//
// Usage:
//   To upload a file to S3:
//      gof3r  --up --file_path=<file_path> --url=<public_url> -h<http_header1> -h<http_header2>...
//   To download a file from S3:
//      gof3r  --down --file_path=<file_path> --url=<public_url>
//
//   The file does not need to be seekable or stat-able.
//
//   Examples:
//     $ gof3r  --up --file_path=test_file --url=https://bucket1.s3.amazonaws.com/object -hx-amz-meta-custom-metadata:123 -hx-amz-meta-custom-metadata2:123abc -hx-amz-server-side-encryption:AES256 -hx-amz-storage-class:STANDARD
//     $ gof3r  --down --file_path=test_file --url=https://bucket1.s3.amazonaws.com/object
//
// Environment:
//
// AwS_ACCESS_KEY – an AWS Access Key Id (required)
//
// AWS_SECRET_KEY – an AWS Secret Access Key (required)
//
// Complete Usage:
//  gof3r [OPTIONS]
//
// Help Options:
//  -h, --help=      Show this help message
//
// Application Options:
//      --up         Upload to S3
//      --down       Download from S3
//  -f, --file_path= canonical path to file
//  -u, --url=       Url of S3 object
//  -h, --headers=   HTTP headers ({})
//  -c, --checksum   Verify integrity with  md5 checksum
package main

import (
	"fmt"
	//"github.com/htcat/htcat"
	"github.com/jessevdk/go-flags"
	"github.com/rlmcpherson/s3/s3util"
	"github.com/rlmcpherson/s3gof3r"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {

	// Parse flags
	args, err := flags.Parse(&opts)
	fmt.Printf(strings.Join(args, " "))

	if err != nil {
		log.Fatal(err)
	}
	s3util.DefaultConfig.AccessKey = os.Getenv("AWS_ACCESS_KEY")
	s3util.DefaultConfig.SecretKey = os.Getenv("AWS_SECRET_KEY")

	start := time.Now()

	if opts.Down && !opts.Up {
		err := s3gof3r.Download(opts.Url, opts.FilePath, opts.Check)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Download completed.")
	} else if opts.Up {
		err := s3gof3r.Upload(opts.Url, opts.FilePath, opts.Header, opts.Check)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Upload completed.")

	} else {
		log.Fatal("specify direction of transfer: up or down")
	}
	log.Println("Duration:", time.Since(start))
	if opts.Debug {
		panic("Dump the stacks.")
	}

}

var opts struct {

	//AccessKey string `short:"k" long:"accesskey" description:"AWS Access Key"`
	//SecretKey string `short:"s" long:"secretkey" description:"AWS Secret Key"`
	Up       bool        `long:"up" description:"Upload to S3"`
	Down     bool        `long:"down" description:"Download from S3"`
	FilePath string      `short:"f" long:"file_path" description:"Path to file. Stdout / Stdin are used if not specified. "`
	Url      string      `short:"u" long:"url" description:"Url of S3 object" required:"true"`
	Header   http.Header `short:"h" long:"headers" description:"HTTP headers"`
	Check    string      `short:"c" long:"md5-checking" description:"Use md5 hash checking to ensure data integrity. Arguments: metadata: calculate md5 before uploading and put in metadata. file: calculate md5 concurrently during upload and store at <url>.md5 Faster than storing in metadata and can be used with pipes." optional:"true" optional-value:"metadata"`
	Debug    bool        `long:"debug" description:"Print debug statements and dump stacks."`
}
