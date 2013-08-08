// Package gof3r is a command-line interface for Amazon AWS S3.
//
// Example usage:
//   To upload a file to S3: 
//      gof3r  --up --file_path=<file_path> --url=<public_url> -h<http_header1> -h<http_header2>...
//   To download a file from S3:
//      gof3r  --down --file_path=<file_path> --url=<public_url> 
//  
//   The file does not need to be seekable or stat-able. 
//
//   Examples:
//   $ gof3r  --up --file_path=test_file --url=https://bucket1.s3.amazonaws.com/object -hx-amz-meta-custom-metadata:123 -hx-amz-meta-custom-metadata2:123abc -hx-amz-server-side-encryption:AES256 -hx-amz-storage-class:STANDARD 
//   $ gof3r  --down --file_path=test_file --url=https://bucket1.s3.amazonaws.com/object 
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

package gof3r

import (
    "github.com/rlmcpherson/s3/s3util"
    "os"
    "io"
    "fmt"
    "net/http"
    "crypto/md5"
)

func Upload(url string, file_path string, header http.Header, check bool) (error) {
    r, err := os.Open(file_path) 
    if err != nil{
        return err
    }
    if (check){
        content_checksum, err := checksum(r)
        if err != nil {
            return err
        }
        header.Add("x-amz-meta-checksum", content_checksum)

    }
    w, err := s3util.Create(url, header, nil) 
    if err != nil {
        return err
    }
    if err := fileCopyClose(w, r); err != nil {return err}
    return nil
}

func Download(url string, file_path string) (error){
    r, err := s3util.Open(url, nil) 
    if err != nil{
        return err
    }
    w, err := os.Create(file_path)
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

func checksum(r io.Reader)(string, error){
    h:= md5.New()
    io.Copy(h, r)
 return fmt.Sprintf("%x", h.Sum(nil)), nil
}