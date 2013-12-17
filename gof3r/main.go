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
// Set AWS keys as environment Variables (required):
//
//  $ export AWS_ACCESS_KEY_ID=<access_key>
//  $ export AWS_SECRET_ACCESS_KEY=<secret_key>
//
// Examples:
//  $ tar -cf - /foo_dir/ | gof3r put -b my_s3_bucket -k bar_dir/s3_object -m x-amz-meta-custom-metadata:abc123 -m x-amz-server-side-encryption:AES256
//  $ gof3r get -b my_s3_bucket -k bar_dir/s3_object | tar -x
//
//
// Complete Usage: get command:
//   gof3r [OPTIONS] get [get-OPTIONS]
//
//   get (download) from S3
//
//   Help Options:
//   -h, --help          Show this help message
//
//   get (download) from S3:
//   -p, --path=         Path to file. Defaults to standard output for streaming. (/dev/stdout)
//   -k, --key=          key of s3 object
//   -b, --bucket=       s3 bucket
//   --md5Check-off      Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently
//                       during puts, stored at <bucket>.md5/<key>.md5, and verified on gets.
//   -c, --concurrency=  Concurrency of transfers (20)
//   -s, --partsize=     initial size of concurrent parts, in bytes (20 MB)
//   --debug             Print debug statements and dump stacks.
//
//   Help Options:
//   -h, --help          Show this help message
//
//
// Complete Usage: put command:
//   gof3r [OPTIONS] put [put-OPTIONS]
//
//   put (upload)to S3
//
//   Help Options:
//     -h, --help          Show this help message
//
//   put (upload) to S3:
//     -p, --path=         Path to file. Defaults to standard input for streaming. (/dev/stdin)
//     -m, --header=       HTTP headers
//     -k, --key=          key of s3 object
//     -b, --bucket=       s3 bucket
//         --md5Check-off  Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently
//                         during puts, stored at <bucket>.md5/<key>.md5, and verified on gets.
//     -c, --concurrency=  Concurrency of transfers (20)
//     -s, --partsize=     initial size of concurrent parts, in bytes (20 MB)
//         --debug         Print debug statements and dump stacks.
//
//   Help Options:
//     -h, --help          Show this help message
package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/rlmcpherson/s3gof3r"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

// Options common to both puts and gets
type CommonOpts struct {
	//Url         string      `short:"u" long:"url" description:"Url of S3 object"` //TODO: bring back url support
	Key          string `long:"key" short:"k" description:"key of s3 object" required:"true"`
	Bucket       string `long:"bucket" short:"b" description:"s3 bucket" required:"true"`
	CheckDisable bool   `long:"md5Check-off" description:"Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently during puts, stored at <bucket>.md5/<key>.md5, and verified on gets."`
	Concurrency  int    `long:"concurrency" short:"c" default:"20" description:"Concurrency of transfers"`
	PartSize     int64  `long:"partsize" short:"s" description:"initial size of concurrent parts, in bytes" default:"20 MB"`
	EndPoint     string `long:"endpoint" description:"Amazon S3 endpoint" default:"s3.amazonaws.com"`
	Debug        bool   `long:"debug" description:"Print debug statements and dump stacks."`
}

var parser = flags.NewParser(nil, flags.Default)

func main() {
	// set the number of processors to use to the number of cpus for parallelization of concurrent transfers
	runtime.GOMAXPROCS(runtime.NumCPU())

	// parser calls the Execute functions on Get and Put, after parsing the command line options.
	start := time.Now()
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
	log.Println("Duration:", time.Since(start))
}

// Uses same environment variables as aws cli
func getKeys() s3gof3r.Keys {
	return s3gof3r.Keys{AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}
}

func debug() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Println("MEMORY STATS")
	log.Println(fmt.Printf("%d,%d,%d,%d\n", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased))
	log.Println("NUM CPU:", runtime.NumCPU())

	//profiling
	f, err := os.Create("memprofileup.out")
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
	time.Sleep(1 * time.Second)
	panic("Dump the stacks:")
}
