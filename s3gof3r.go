// Package s3gof3r is a command-line interface for Amazon AWS S3.
//
package s3gof3r

import (
	"crypto/md5"
	"fmt"
	"hash"
	//"github.com/op/go-logging"
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

func Upload(url string, file_path string, header http.Header, check string) error {

	var md5Hash hash.Hash

	r, err := os.Open(file_path)
	if err != nil {
		if file_path == "" {
			r = os.Stdin
		} else {
			return err
		}
	}

	//br := bufio.NewReader(r)
	defer r.Close()

	log.Println("Check option: ", check)

	if check == "metadata" {
		// precalculate md5 for http header
		md5Hash, err := md5Calc(r)
		if err != nil {
			return err
		}
		if header == nil {
			header = make(http.Header)
		}

		md5Header := fmt.Sprintf("%x", md5Hash.Sum(nil))
		header.Set(checkSumHeader, md5Header)
		log.Println("POST REQ HEADER:")
		header.Write(os.Stderr)
	}

	w, err := s3util.Create(url, header, nil)
	if err != nil {
		return err
	}
	mw := io.MultiWriter(w)

	if check == "file" {
		md5Hash = md5.New()
		mw = io.MultiWriter(md5Hash, w)

	}
	// buffered io to reduce disk IO
	//bmw := bufio.NewWriter(mw)

	if _, err := io.Copy(mw, r); err != nil {
		return err
	}

	//if err := bmw.Flush(); err != nil {
	//return err
	//}

	if err := w.Close(); err != nil {
		return err
	}

	// Write md5 to file and upload
	if check == "file" {
		if err := md5FileUpload(md5Hash, url); err != nil {
			return err
		}
	}

	return nil
}

func Download(url string, file_path string, check string) error {
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
	bw := bufio.NewWriter(w)

	if check == "metadata" {
		h := md5.New()

		// buffered to reduce disk IO
		bh := bufio.NewWriter(h)
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
		// TODO: download md5 file
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
		if _, err := io.Copy(bw, r); err != nil {
			return err
		}

	}
	return nil
}

func md5Calc(r io.ReadSeeker) (hash.Hash, error) {
	log.Println("Calculating MD5 Hash...")
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	if _, err := r.Seek(0, 0); err != nil {
		return nil, err
	}
	return h, nil
}

func md5FileUpload(h hash.Hash, url string) error {

	md5Url := url + ".md5"
	md5 := fmt.Sprintf("%x", h.Sum(nil))
	w, err := s3util.Create(md5Url, nil, nil)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, md5); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	log.Println(md5Url, " uploaded: ", md5)
	return nil
}
