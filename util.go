package s3gof3r

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// convenience multipliers
const (
	_        = iota
	kb int64 = 1 << (10 * iota)
	mb
	gb
	tb
	pb
	eb
)

// Min and Max functions
func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Error type and functions for http response
// http://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html
type respError struct {
	r         *http.Response
	Code      string
	Message   string
	Resource  string
	RequestId string
}

func newRespError(r *http.Response) *respError {
	e := new(respError)
	e.r = r
	b, _ := ioutil.ReadAll(r.Body)
	xml.NewDecoder(bytes.NewReader(b)).Decode(e) // parse error from response
	r.Body.Close()
	return e
}

func (e *respError) Error() string {
	return fmt.Sprintf(
		"Error:  %d: %q",
		e.r.StatusCode,
		e.Message,
	)
}

func md5Check(r io.ReadSeeker, given string) (err error) {
	h := md5.New()
	if _, err = io.Copy(h, r); err != nil {
		return
	}
	if _, err = r.Seek(0, 0); err != nil {
		return
	}
	calculated := fmt.Sprintf("%x", h.Sum(nil))
	if calculated != given {
		log.Println(base64.StdEncoding.EncodeToString(h.Sum(nil)))
		return fmt.Errorf("md5 mismatch. given:%s calculated:%s", given, calculated)
	}
	return nil
}

func bucketFromUrl(subdomain string) string {
	s := strings.Split(subdomain, ".")
	return strings.Join(s[:len(s)-1], ".")
}

func checkClose(c io.Closer, err *error) {
	cerr := c.Close()
	if *err == nil {
		*err = cerr
	}
}
