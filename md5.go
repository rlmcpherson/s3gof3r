package s3gof3r

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	checkSumHeader = "x-amz-meta-md5-hash"
)

func Upload(key string, bucket string, file_path string, header http.Header, check string, conc int) error {

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
	s3 := New("", k)
	b := s3.Bucket(bucket)
	c := DefaultConfig
	c.Concurrency = conc
	w, err := b.PutWriter(key, header, c)

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
		if err := md5FileUpload(md5Hash, key, b); err != nil {
			return err
		}
	}

	return nil
}

func Download(key string, bucket string, file_path string, check string, conc int) error {
	var r io.ReadCloser

	k := Keys{AccessKey: os.Getenv("AWS_ACCESS_KEY"),
		SecretKey: os.Getenv("AWS_SECRET_KEY"),
	}
	s3 := New("", k)
	b := s3.Bucket(bucket)
	c := DefaultConfig
	c.Concurrency = conc
	r, header, err := b.GetReader(key, c)
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

	if _, err := io.Copy(mw, r); err != nil {
		return err
	}
	// flush buffers to ensure all data is copied
	bw.Flush()
	bh.Flush()
	calculatedHash := fmt.Sprintf("%x", h.Sum(nil))
	log.Println("Calculated MD5 Hash:", calculatedHash)
	log.Println("Check option: ", check)
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
			remoteHash, err = md5fileDownload(key, b)
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

func md5FileUpload(h hash.Hash, key string, b *Bucket) error {

	md5 := fmt.Sprintf("%x", h.Sum(nil))
	c := DefaultConfig
	path := "/.md5" + key + ".md5"
	w, err := b.PutWriter(path, nil, c)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, md5); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return nil
}

func md5fileDownload(key string, b *Bucket) (string, error) {

	path := "/.md5" + key + ".md5"
	r, _, err := b.GetReader(path, nil)
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
