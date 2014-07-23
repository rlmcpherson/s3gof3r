// gof3r is a command-line interface for s3gof3r: fast, concurrent, streaming access to Amazon S3.
//
// Example Usage:
//   To stream up to S3:
//      $  <input_stream> | gof3r put -b <bucket> -k <s3_path>
//   To stream down from S3:
//      $ gof3r get -b <bucket> -k <s3_path> | <output_stream>
//   To upload a file to S3:
//      $ gof3r cp <local_path> s3://<bucket>/<s3_path> -m<http_header1> -m<http_header2>...
//   To download a file from S3:
//      $ gof3r cp s3://<bucket>/<s3_path> <local_path>
//
//
// Set AWS keys as environment Variables (required unless using ec2 instance-based credentials):
//
//  $ export AWS_ACCESS_KEY_ID=<access_key>
//  $ export AWS_SECRET_ACCESS_KEY=<secret_key>
//
// Examples:
//  $ tar -cf - /foo_dir/ | gof3r put -b my_s3_bucket -k bar_dir/s3_object -m x-amz-meta-custom-metadata:abc123 -m x-amz-server-side-encryption:AES256
//  $ gof3r get -b my_s3_bucket -k bar_dir/s3_object | tar -x
//
//
// MAN PAGE
//
// http://randallmcpherson.com/s3gof3r/gof3r/gof3r.html
//
// A man page may also be generated with `gof3r -m`
//
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/rlmcpherson/s3gof3r"
)

const (
	name    = "gof3r"
	version = "0.4.1"
)

var AppOpts struct {
	Version func() `long:"version" short:"v" description:"Print version"`
	Man     func() `long:"manpage" short:"m" description:"Create gof3r.man man page in current directory"`
}

var parser = flags.NewParser(&AppOpts, (flags.HelpFlag | flags.PassDoubleDash))

func init() {
	// set the number of processors to use to the number of cpus for parallelization of concurrent transfers
	runtime.GOMAXPROCS(runtime.NumCPU())

	// set parser fields
	parser.ShortDescription = "streaming, concurrent s3 client"

	AppOpts.Version = func() {
		fmt.Fprintf(os.Stderr, "%s version %s\n", name, version)
		os.Exit(0)
	}

	AppOpts.Man = func() {
		f, err := os.Create(name + ".man")
		if err != nil {
			log.Fatal(err)
		}
		parser.WriteManPage(f)
		fmt.Fprintf(os.Stderr, "man page written to %s\n", f.Name())
		os.Exit(0)
	}
}

// CommonOpts are Options common to both puts and gets
type CommonOpts struct {
	//Url         string      `short:"u" long:"url" description:"URL of S3 object"` //TODO: bring back url support
	Key         string `long:"key" short:"k" description:"S3 object key" required:"true"`
	Bucket      string `long:"bucket" short:"b" description:"S3 bucket" required:"true"`
	NoSSL       bool   `long:"no-ssl" description:"Do not use SSL for endpoint connection."`
	NoMd5       bool   `long:"no-md5" description:"Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently during puts, stored at <bucket>.md5/<key>.md5, and verified on gets."`
	Concurrency int    `long:"concurrency" short:"c" default:"10" description:"Concurrency of transfers"`
	PartSize    int64  `long:"partsize" short:"s" description:"Initial size of concurrent parts, in bytes" default:"20971520"`
	EndPoint    string `long:"endpoint" description:"Amazon S3 endpoint" default:"s3.amazonaws.com"`
	Debug       bool   `long:"debug" description:"Enable debug logging."`
}

func main() {
	// parser calls the Execute functions on Get and Put, after parsing the command line options.
	start := time.Now()
	if _, err := parser.Parse(); err != nil {

		// handling for flag parse errors
		if ferr, ok := err.(*flags.Error); ok {
			if ferr.Type == flags.ErrHelp {
				parser.WriteHelp(os.Stderr)
			} else {
				var cmd string
				if parser.Active != nil {
					cmd = parser.Active.Name
				}
				fmt.Fprintf(os.Stderr, "gof3r error: %s\n", err)
				fmt.Fprintf(os.Stderr, "run 'gof3r %s --help' for usage.\n", cmd)
			}
		} else { // handle non-parse errors
			fmt.Fprintf(os.Stderr, "gof3r error: %s\n", err)
		}
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Duration: %v\n", time.Since(start))
}

// Gets the AWS Keys from environment variables or the instance-based metadata on EC2
// Environment variables are attempted first, followed by the instance-based credentials.
// It returns an error if no keys are found.
func getAWSKeys() (keys s3gof3r.Keys, err error) {

	keys, err = s3gof3r.EnvKeys()
	if err == nil {
		return
	}
	keys, err = s3gof3r.InstanceKeys()
	if err == nil {
		return
	}
	err = errors.New("no AWS keys found")
	return
}

func debug() {
	log.Println("Running debug report...")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Println("MEMORY STATS")
	log.Printf("%d,%d,%d,%d\n", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased)
}
