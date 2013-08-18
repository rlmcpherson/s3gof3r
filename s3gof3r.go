// Package s3gof3r is a command-line interface for Amazon AWS S3.
//
package s3gof3r

import (
	"crypto/md5"
	"fmt"
	//"github.com/op/go-logging"
	//	"archive/tar"
	"bufio"
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
	br := bufio.NewReader(r)
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
		header.Write(os.Stderr)
	}
	w, err := s3util.Create(url, header, nil)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, br); err != nil {
		return err
	}

	err = w.Close()
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
		if file_path == "" {
			w = os.Stdout
		} else {
			return err
		}
	}
	defer w.Close()

	if check {
		h := md5.New()

		// buffered to reduce disk IO
		bh := bufio.NewWriter(h)
		bw := bufio.NewWriter(w)
		mw := io.MultiWriter(bw, bh)
		if _, err := io.Copy(mw, r); err != nil {
			return err
		}
		// flush buffers to ensure all data is copied
		bw.Flush()
		bh.Flush()

		calculatedHash := fmt.Sprintf("%x", h.Sum(nil))
		log.Println("Calculated MD5 Hash:", calculatedHash)
		remoteHash := header.Get(checkSumHeader)
		log.Println("GET REQ HEADER:")
		header.Write(os.Stderr)
		if remoteHash == "" {
			return fmt.Errorf("Could not verify content. Http header %s not found.", checkSumHeader)
		}

		if remoteHash != calculatedHash {
			return fmt.Errorf("MD5 hash comparison failed for file %s. Hash from header: %s."+
				"Calculated hash: %s.", file_path, remoteHash, calculatedHash)
		}
	} else {
		if _, err := io.Copy(w, r); err != nil {
			return err
		}

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
