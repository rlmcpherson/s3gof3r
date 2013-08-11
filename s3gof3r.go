// Package s3gof3r is a command-line interface for Amazon AWS S3.
//
package s3gof3r

import (
	"crypto/md5"
	//"encoding/hex"
	"fmt"
	"github.com/rlmcpherson/s3/s3util"
	"io"
	"net/http"
	"os"
)

func Upload(url string, file_path string, header http.Header, check bool) error {
	r, err := os.Open(file_path)
	if err != nil {
		return err
	}
	if check {
		md5hash, err := md5hash(io.ReadSeeker(r))
		if err != nil {
			return err
		}
		fmt.Println(md5hash)
		header.Add("x-amz-meta-md5-hash", md5hash)
		header.Write(os.Stdout)

	}
	w, err := s3util.Create(url, header, nil)
	if err != nil {
		return err
	}
	if err := fileCopyClose(w, r); err != nil {
		return err
	}
	return nil
}

func Download(url string, file_path string, check bool) error {
	r, header, err := s3util.Open(url, nil)
	if err != nil {
		return err
	}
	w, err := os.Create(file_path)
	if err != nil {
		return err
	}
	if check {
		remoteHash := header.Get("x-amz-meta-md5-hash")
		if remoteHash == "" {
			return fmt.Errorf("Could not verify content. Http header 'Md5-Hash' header not found.")
		}

		//calculatedHash, err := md5hash(io.ReadSeeker(r.Read))
		if err != nil {
			return err
		}
		fmt.Println(md5hash)
		header.Write(os.Stdout)

	}

	if err := fileCopyClose(w, r); err != nil {
		return err
	}
	return nil
}

func fileCopyClose(w io.WriteCloser, r io.ReadCloser) error {
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func md5hash(r io.ReadSeeker) (string, error) {
	h := md5.New()
	io.Copy(h, r)
	r.Seek(0, 0)
	//encoder := base64.NewEncoder(base64.StdEncoding, b64)
	return (fmt.Sprintf("%x", h.Sum(nil))), nil
	//return hex.EncodeToString(h.Sum(nil)), nil
}
