package main

import (
    "github.com/rlmcpherson/gof3r"
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
        err := gof3r.Download(opts.Url, opts.FilePath)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
        }
    }    else if opts.Up{
        err := gof3r.Upload(opts.Url, opts.FilePath, opts.Header, opts.Check)
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
