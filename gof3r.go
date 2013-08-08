// Package gof3r is a command-line interface for Amazon AWS S3.
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