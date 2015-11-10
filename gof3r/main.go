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
	"io"
	"os"
	"runtime"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/rlmcpherson/s3gof3r"
)

const (
	name    = "gof3r"
	version = "0.5.0"
)

func main() {
	// set the number of processors to use to the number of cpus for parallelization of concurrent transfers
	runtime.GOMAXPROCS(runtime.NumCPU())

	start := time.Now()

	// parse ini file
	if err := parseIni(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	// parser calls the Execute function for the command after parsing the command line options.
	if _, err := parser.Parse(); err != nil {

		if appOpts.WriteIni {
			writeIni() // exits
		}

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
	fmt.Fprintf(os.Stderr, "duration: %v\n", time.Since(start))
}

// getAWSKeys gets the AWS Keys from environment variables or the instance-based metadata on EC2
// Environment variables are attempted first, followed by the instance-based credentials.
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

func checkClose(c io.Closer, err error) {
	if c != nil {
		cerr := c.Close()
		if err == nil {
			err = cerr
		}
	}
}
