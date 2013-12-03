// Command gof3r is a command-line interface for s3gof3r: fast, concurrent, streaming access to Amazon S3.
//
// Usage:
//   To upload a file to S3:
//      gof3r  put --buckt=<bucket> --key=<path> -h<http_header1> -h<http_header2>...
//   To download a file from S3:
//      gof3r  --down --file_path=<file_path> --url=<public_url>
//
//
//   Examples:
//     $ gof3r  --up --file_path=test_file --url=https://bucket1.s3.amazonaws.com/object -hx-amz-meta-custom-metadata:123 -hx-amz-meta-custom-metadata2:123abc -hx-amz-server-side-encryption:AES256 -hx-amz-storage-class:STANDARD
//     $ gof3r  --down --file_path=test_file --url=https://bucket1.s3.amazonaws.com/object
//
// Set Environment Variables:
//
//  $ export AWS_ACCESS_KEY_ID=<access_key>
//  $ export AWS_SECRET_ACCESS_KEY=<secret_key>
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
//  -c, --md5-checking   Verify integrity with  md5 checksum
package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/rlmcpherson/s3gof3r"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

// Options common to both puts and gets
type CommonOpts struct {
	//Url         string      `short:"u" long:"url" description:"Url of S3 object"` //TODO: bring back url support
	Key          string      `long:"key" description:"key of s3 object" required:"true"`
	Bucket       string      `long:"bucket" description:"s3 bucket" required:"true"`
	Header       http.Header `long:"header" short:"m" description:"HTTP headers"`
	CheckDisable bool        `long:"md5Check-off" description:"Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently during puts, stored at <bucket>.md5/<key>.md5, and verified on gets."`
	Concurrency  int         `long:"concurrency" short:"c" default:"20" description:"Concurrency of transfers"`
	PartSize     int64       `long:"partsize" short:"s" description:"initial size of concurrent parts, in bytes" default:"20 MB"`
	Debug        bool        `long:"debug" description:"Print debug statements and dump stacks."`
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
