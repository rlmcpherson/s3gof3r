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

    if opts.Down && !opts.Up{
        err := download(opts.Url)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
        }
    }    else if opts.Up{
        err := upload(opts.Url, opts.Header)
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
    Up bool `long:"up" description:"Upload to S3"`
    Down bool `long:"down" description:"Download from S3"`
    FilePath string `short:"f" long:"file_path" description:"canonical path to file" required:"true"`
    Url string `short:"u" long:"url" description:"Url of S3 object" required:"true"` 
    Header http.Header `short:"h" long:"headers" description:"HTTP headers"` 

}

//func open(opts struct) (io.ReadCloser, error){
//    return os.Open(opts.FilePath)
//}



func upload(url string, header http.Header) (error) {
    r, err := os.Open(opts.FilePath) 
    if err != nil{
        return err
    }
    w, err := s3util.Create(url, header, nil) 
    if err != nil {
        return err
    }
    if err := fileCopyClose(w, r); err != nil {return err}
    return nil
}

func download(url string) (error){
    r, err := s3util.Open(opts.Url, nil) 
    if err != nil{
        return err
    }
    w, err := os.Create(opts.FilePath)
    if err != nil{
        return err
    }
    if err := fileCopyClose(w, r); err != nil {return err}
    return nil
}

func fileCopyClose(w io.WriteCloser, r io.ReadCloser) (error){
    if _, err := io.Copy(w,r); err != nil {return err}
    if err := w.Close() ; err != nil {return err }
return nil
}