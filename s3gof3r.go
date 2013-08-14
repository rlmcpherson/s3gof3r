// Package s3gof3r is a command-line interface for Amazon AWS S3.
//
package s3gof3r

import (
	"crypto/md5"
	"fmt"
	//"github.com/op/go-logging"
	"github.com/rlmcpherson/s3/s3util"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	checkSumHeader = "x-amz-meta-md5-hash"
)

func Upload(url string, file_path string, header http.Header, check bool) error {
	r, err := os.Open(file_path)
	if err != nil {
		return err
	}
	defer r.Close()
	if check {
		md5hash, err := md5hash(file_path)
		if err != nil {
			return err
		}
		if header == nil {
			header = make(http.Header)
		}
		header.Set(checkSumHeader, md5hash)
		log.Println("POST REQ HEADER:")
		header.Write(os.Stdout)
	}
	w, err := s3util.Create(url, header, nil)
	if err != nil {
		return err
	}
	defer w.Close()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}

func Download(url string, file_path string, check bool) error {
	r, header, err := s3util.Open(url, nil)
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := os.Create(file_path)
	if err != nil {
		return err
	}
	defer w.Close()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	if check {
		remoteHash := header.Get(checkSumHeader)
		if remoteHash == "" {
			return fmt.Errorf("Could not verify content. Http header %s not found.", checkSumHeader)
		}
		calculatedHash, err := md5hash(file_path)
		if err != nil {
			return err
		}
		if remoteHash != calculatedHash {
			return fmt.Errorf("MD5 hash comparison failed for file %s. Hash from header: %s."+
				"Calculated hash: %s.", file_path, remoteHash, calculatedHash)
		}
		log.Println("GET REQ HEADER:")
		header.Write(os.Stdout)
	}
	return nil
}

func md5hash(file_path string) (string, error) {
	log.Println("Calculating MD5 Hash...")
	r, err := os.Open(file_path)
	defer r.Close()
	if err != nil {
		return "", err
	}
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return (fmt.Sprintf("%x", h.Sum(nil))), nil
}
