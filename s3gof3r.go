// Package s3gof3r is a command-line interface for Amazon AWS S3.
//
package s3gof3r

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"github.com/rlmcpherson/s3/s3util"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	url_ "net/url"
	"os"
	"time"
)

const (
	checkSumHeader = "x-amz-meta-md5-hash"
)

func Upload(url string, file_path string, header http.Header, check string, conc int) error {

	var md5Hash hash.Hash

	r, err := os.Open(file_path)
	if err != nil {
		if file_path == "" {
			r = os.Stdin
		} else {
			return err
		}
	}

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
	k := Keys{AccessKey: os.Getenv("AWS_ACCESS_KEY"),
		SecretKey: os.Getenv("AWS_SECRET_KEY"),
	}
	s3_ := New("", k)
	b := s3_.Bucket("rm-dev-repos")
	key, _ := url_.Parse(url)
	path := key.Path
	c := DefaultConfig
	c.Concurrency = conc
	w, err := b.PutWriter(path, header, c)

	if err != nil {
		return err
	}
	mw := io.MultiWriter(w)

	if check == "file" {
		md5Hash = md5.New()
		mw = io.MultiWriter(md5Hash, w)

	}

	if _, err := io.Copy(mw, r); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	// Write md5 to file and upload
	if check == "file" {
		if err := md5FileUpload(md5Hash, url, b); err != nil {
			return err
		}
	}

	return nil
}

func Download(url string, file_path string, check string, conc int) error {
	//r, header, err := s3util.Open(url, s3Config())
	var r io.ReadCloser

	k := Keys{AccessKey: os.Getenv("AWS_ACCESS_KEY"),
		SecretKey: os.Getenv("AWS_SECRET_KEY"),
	}
	s3_ := New("", k)
	b := s3_.Bucket("rm-dev-repos")
	key, _ := url_.Parse(url)
	path := key.Path
	c := DefaultConfig
	c.Concurrency = conc
	r, header, err := b.GetReader(path, c)
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

	// Calculate md5 hash concurrently
	h := md5.New()
	// buffered to reduce disk IO
	bw := bufio.NewWriter(w)
	bh := bufio.NewWriter(h)
	mw := io.MultiWriter(bw, bh)

	log.Println("Starting copy from s3Get")
	if _, err := io.Copy(mw, r); err != nil {
		return err
	}
	// flush buffers to ensure all data is copied
	bw.Flush()
	bh.Flush()
	calculatedHash := fmt.Sprintf("%x", h.Sum(nil))
	log.Println("Calculated MD5 Hash:", calculatedHash)

	if check != "" {

		remoteHash := header.Get(checkSumHeader)
		if check == "metadata" {

			log.Println("GET REQ HEADER:")
			header.Write(os.Stderr)
			if remoteHash == "" {
				return fmt.Errorf("Could not checksum content. Http header %s not found.", checkSumHeader)
			}
		} else { // check == file
			// download <url>.md5 file
			remoteHash, err = md5fileDownload(url, b)
			if err != nil {
				return fmt.Errorf("Could not checksum content: %s", err)

			}

		}

		if remoteHash != calculatedHash {
			return fmt.Errorf("MD5 checksums do not match. Given: %s."+
				"Calculated: %s.", remoteHash, calculatedHash)
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

func md5FileUpload(h hash.Hash, url string, b *Bucket) error {

	md5Path, err := md5Path(url)
	if err != nil {
		return err
	}
	md5 := fmt.Sprintf("%x", h.Sum(nil))
	w, err := b.PutWriter(md5Path, nil, nil)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, md5); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	log.Println(md5Path, " uploaded: ", md5)
	return nil
}

func md5fileDownload(url string, b *Bucket) (string, error) {

	md5Path, err := md5Path(url)
	if err != nil {
		return "", err
	}
	r, _, err := b.GetReader(md5Path, nil)
	if err != nil {
		return "", err
	}

	md5, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	log.Println("Md5 file downloaded:", string(md5))
	return string(md5), nil
}

// Calculate url for md5 file in subdirectory of bucket / directory where the file is stored
// e.g. the md5 for https://mybucket.s3.amazonaws.com/gof3r will be stored in
// https://mybucket.s3.amazonaws.com/.md5/gof3r.md5
func md5Path(fileUrl string) (string, error) {

	parsed_url, err := url_.Parse(fileUrl)
	if err != nil {
		return "", err
	}
	path := parsed_url.Path
	parsed_url.Path = ""
	return fmt.Sprint("/.md5", path, ".md5"), nil

}

func s3Config() (config *s3util.Config) {
	config = s3util.DefaultConfig
	config.AccessKey = os.Getenv("AWS_ACCESS_KEY")
	config.SecretKey = os.Getenv("AWS_SECRET_KEY")
	config.Client = createClientWithTimeout(1 * time.Second)
	config.Concurrency = 20

	return config

}
