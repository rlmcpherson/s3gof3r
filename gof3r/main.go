// Command gof3r is a command-line interface for s3gof3r: fast, concurrent, streaming access to Amazon S3.
//
// Usage:
//   To stream up to S3:
//      $  <input_stream> | gof3r put -b <bucket> -k <s3_path>
//   To stream down from S3:
//      $ gof3r get -b <bucket> -k <s3_path> | <output_stream>
//   To upload a file to S3:
//      $ gof3r  put --path=<local_path> --bucket=<bucket> --key=<s3_path> -m<http_header1> -m<http_header2>...
//   To download a file from S3:
//      $ gof3r  get --bucket=<bucket> --key=<path>
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
// FULL USAGE
//
//    get
//        download from S3
//
//        get (download) object from S3
//
//        -p, --path
//               Path to file. Defaults to standard output for streaming.
//
//        -k, --key
//               S3 object key
//
//        -b, --bucket
//               S3 bucket
//
//        --no-ssl
//               Do not use SSL for endpoint connection.
//
//        --no-md5
//               Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently during puts, stored at <bucket>.md5/<key>.md5, and verified on gets.
//
//        -c, --concurrency
//               Concurrency of transfers
//
//        -s, --partsize
//               Initial size of concurrent parts, in bytes
//
//        --endpoint
//               Amazon S3 endpoint
//
//        --debug
//               Enable debug logging.
//
//        -v, --versionId
//               Version ID of the object. Incompatible with md5 check (use --no-md5).
//
//        -h, --help
//               Show this help message
//
//    put
//        upload to S3
//
//        put (upload) data to S3 object
//
//        -p, --path
//               Path to file. Defaults to standard input for streaming.
//
//        -k, --key
//               S3 object key
//
//        -b, --bucket
//               S3 bucket
//
//        --no-ssl
//               Do not use SSL for endpoint connection.
//
//        --no-md5
//               Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently during puts, stored at <bucket>.md5/<key>.md5, and verified on gets.
//
//        -c, --concurrency
//               Concurrency of transfers
//
//        -s, --partsize
//               Initial size of concurrent parts, in bytes
//
//        --endpoint
//               Amazon S3 endpoint
//
//        --debug
//               Enable debug logging.
//
//        -m, --header
//               HTTP headers
//
//        -h, --help
//               Show this help message
//
//
//
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/rlmcpherson/s3gof3r"
)

const (
	name    = "gof3r"
	version = "0.3.3"
)

var AppOpts struct {
	Version func() `long:"version" short:"v" description:"Print version"`
	Man     func() `long:"manpage" short:"m" description:"Create gof3r.man man page in current directory"`
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

var parser = flags.NewParser(&AppOpts, (flags.HelpFlag | flags.PassDoubleDash))

func main() {
	// set the number of processors to use to the number of cpus for parallelization of concurrent transfers
	runtime.GOMAXPROCS(runtime.NumCPU())

	AppOpts.Version = func() {
		fmt.Fprintf(os.Stderr, "%s version %s\n", name, version)
		os.Exit(0)
	}

	AppOpts.Man = func() {
		f, err := os.Create("gof3r.man")
		if err != nil {
			log.Fatal(err)
		}
		parser.WriteManPage(f)
		os.Exit(0)
	}

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
	log.Println("Duration:", time.Since(start))
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
	err = errors.New("No AWS Keys found.")
	return
}

func debug() {
	log.Println("Running debug report...")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Println("MEMORY STATS")
	log.Printf("%d,%d,%d,%d\n", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased)
	log.Println("NUM CPU:", runtime.NumCPU())

	//profiling
	f, err := os.Create("memprofileup.out")
	defer f.Close()
	fg, err := os.Create("goprof.out")
	fb, err := os.Create("blockprof.out")
	if err != nil {
		log.Fatal(err)
	}
	pprof.WriteHeapProfile(f)
	pprof.Lookup("goroutine").WriteTo(fg, 0)
	pprof.Lookup("block").WriteTo(fb, 0)
	f.Close()
	fg.Close()
	fb.Close()
}
