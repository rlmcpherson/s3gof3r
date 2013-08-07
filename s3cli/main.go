// Command s3cli provides a command line interface for Amazon S3 uploads and downloads.
// Usage:
//  To upload a file to S3: 
//      s3cli  --up --file_path=<file_path> --url=<public_url> -h<http_header1> -h<http_header2>...
//  To download a file from S3:
//      s3cli  --down --file_path=<file_path> --url=<public_url> 
//  
//  The file does not need to be seekable or stat-able. 
//
//  Examples:
//  $ s3cli  --up --file_path=test_file --url=https://bucket1.s3.amazonaws.com/object -hx-amz-meta-custom-metadata:123 -hx-amz-meta-custom-metadata2:123abc -hx-amz-server-side-encryption:AES256 -hx-amz-storage-class:STANDARD 
//  $ s3cli  --down --file_path=test_file --url=https://bucket1.s3.amazonaws.com/object 
//
// Environment:
//
// AwS_ACCESS_KEY – an AWS Access Key Id (required)
//
// AWS_SECRET_KEY – an AWS Secret Access Key (required)
//
// Complete Usage:
//  s3cli [OPTIONS]
//
//Help Options:
//  -h, --help=      Show this help message
//
//Application Options:
//      --up         Upload to S3
//      --down       Download from S3
//  -f, --file_path= canonical path to file
//  -u, --url=       Url of S3 object
//  -h, --headers=   HTTP headers ({})
//  -c, --checksum   Verify integrity with  md5 checksum

package main

import (
    "github.com/rlmcpherson/s3cli"
    "github.com/rlmcpherson/s3/s3util"
    "os"
    "fmt"
    "strings"
    "github.com/jessevdk/go-flags"
    "net/http"
    "log"

)

func main() {

    // Parse flags
    args, err := flags.Parse(&opts)
    fmt.Printf( strings.Join(args, " "))

    if err != nil {
        os.Exit(1)

    }
    s3util.DefaultConfig.AccessKey = os.Getenv("AWS_ACCESS_KEY")
    s3util.DefaultConfig.SecretKey = os.Getenv("AWS_SECRET_KEY")

    if opts.Down && !opts.Up{
        err := s3cli.Download(opts.Url, opts.FilePath)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
        }
    }    else if opts.Up{
        err := s3cli.Upload(opts.Url, opts.FilePath, opts.Header, opts.Check)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
        }

    } else{
        log.Fatal("specify direction of transfer: up or down")
    }

}

var opts struct {

    //AccessKey string `short:"k" long:"accesskey" description:"AWS Access Key"`
    //SecretKey string `short:"s" long:"secretkey" description:"AWS Secret Key"`
    Up bool `long:"up" description:"Upload to S3"`
    Down bool `long:"down" description:"Download from S3"`
    FilePath string `short:"f" long:"file_path" description:"canonical path to file" required:"true"`
    Url string `short:"u" long:"url" description:"Url of S3 object" required:"true"` 
    Header http.Header `short:"h" long:"headers" description:"HTTP headers"` 
    Check bool `short:"c" long:"checksum" description:"Verify integrity with  md5 checksum"`

}
