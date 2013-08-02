package main

import (
    "github.com/rlmcpherson/s3/s3util"
    "os"
    "io"
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
    s3util.DefaultConfig.AccessKey = os.Getenv("S3_ACCESS_KEY")
    s3util.DefaultConfig.SecretKey = os.Getenv("S3_SECRET_KEY")

    if opts.Down{
        r, err := download(opts.Url)
        w, _ := os.Create(opts.FilePath)
        io.Copy(w, r)
        w.Close()
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
        }
    }    else if opts.Up{
        r, _ := os.Open(opts.FilePath)
        w, err := upload(opts.Url, opts.Header)
        io.Copy(w,r)
        w.Close()
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
        }

    } else{
        log.Fatal("specify direction of transfer: up or down")
    }

}

var opts struct {

    //AccessKey string `short:"k" long:"accesskey" description:"AWS Access Key" required:"true"`
    //SecretKey string `short:"s" long:"secretkey" description:"AWS Secret Key" required:"true"`
    //Action Action `short:"a" long:"action" description:"direction of data transfer" required:"true"`
    Up bool `long:"up" description:"Upload to S3"`
    Down bool `long:"down" description:"Download from S3"`
    FilePath string `short:"f" long:"file_path" description:"canonical path to file" required:"true"`
    Url string `short:"u" long:"url" description:"Url of S3 object" required:"true"` 
    Header http.Header `short:"h" long:"headers" description:"HTTP headers"` 

}

//func open(opts struct) (io.ReadCloser, error){
//    return os.Open(opts.FilePath)
//}



func upload(url string, header http.Header) (io.WriteCloser, error) {
    return  s3util.Create(url, header, nil)
}

func download(url string) (io.ReadCloser, error){
    return s3util.Open(opts.Url, nil) 
}