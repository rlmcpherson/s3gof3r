package s3gof3r

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"io/ioutil"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// See Amazon S3 Developer Guide for explanation
// http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html
var paramsToSign = map[string]bool{
	"acl":                          true,
	"location":                     true,
	"logging":                      true,
	"notification":                 true,
	"partNumber":                   true,
	"policy":                       true,
	"requestPayment":               true,
	"torrent":                      true,
	"uploadId":                     true,
	"uploads":                      true,
	"versionId":                    true,
	"versioning":                   true,
	"versions":                     true,
	"response-content-type":        true,
	"response-content-language":    true,
	"response-expires":             true,
	"response-cache-control":       true,
	"response-content-disposition": true,
	"response-content-encoding":    true,
}

func (b *Bucket) Sign(req *http.Request) {
	if dateHeader := req.Header.Get("Date"); dateHeader == "" {
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}
	if b.S3.Keys.SecurityToken != "" {
		req.Header.Set("X-Amz-Security-Token", b.S3.Keys.SecurityToken)
	}
	hm := hmac.New(sha1.New, []byte(b.S3.Keys.SecretKey))
	b.writeSignature(hm, req)
	signature := make([]byte, base64.StdEncoding.EncodedLen(hm.Size()))
	base64.StdEncoding.Encode(signature, hm.Sum(nil))
	req.Header.Set("Authorization", "AWS "+b.S3.Keys.AccessKey+":"+string(signature))
}

func (b *Bucket) Create(accessLevel string, force bool, c *Config) error {
	if c == nil {
		c = DefaultConfig
	}

	if accessLevel == "" {
		accessLevel = "private"
	}

	url := b.Url("", c)
	req, err := http.NewRequest("PUT", url.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Host", url.Host)
	req.Header.Set("x-amz-acl", accessLevel)
	b.Sign(req)
	resp, err :=  c.Client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode <= 200 && resp.StatusCode <= 206 {
		return nil
	} else {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if !force && resp.StatusCode == 409 && strings.Contains(string(bodyBytes), "BucketAlreadyOwnedByYou") {
			return nil
		}
		return fmt.Errorf("Bucket creation failed with `%d` status code & `%s` response body", resp.StatusCode, bodyBytes)
	}
}

// From Amazon API documentation:
//
// Signature = Base64( HMAC-SHA1( YourSecretAccessKeyID, UTF-8-Encoding-Of( StringToSign ) ) );
//
// StringToSign = HTTP-Verb + "\n" +
//   Content-MD5 + "\n" +
//   Content-Type + "\n" +
//   Date + "\n" +
//   CanonicalizedAmzHeaders +
//   CanonicalizedResource;
func (b *Bucket) writeSignature(w io.Writer, r *http.Request) {
	w.Write([]byte(r.Method))
	w.Write([]byte{'\n'})
	w.Write([]byte(r.Header.Get("content-md5")))
	w.Write([]byte{'\n'})
	w.Write([]byte(r.Header.Get("content-type")))
	w.Write([]byte{'\n'})
	if _, ok := r.Header["X-Amz-Date"]; !ok {
		w.Write([]byte(r.Header.Get("date")))
	}
	r.Header.Set("User-Agent", "S3Gof3r")
	w.Write([]byte{'\n'})
	b.writeCanonicalizedAmzHeaders(w, r)
	b.writeCanonicializedResource(w, r)
}

// See Amazon S3 Developer Guide for explanation
// http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html
func (b *Bucket) writeCanonicalizedAmzHeaders(w io.Writer, r *http.Request) {
	var amzHeaders []string

	for h, _ := range r.Header {
		if strings.HasPrefix(strings.ToLower(h), "x-amz-") {
			amzHeaders = append(amzHeaders, h)
		}
	}
	sort.Strings(amzHeaders)
	for _, h := range amzHeaders {
		v := r.Header[h]
		w.Write([]byte(strings.ToLower(h)))
		w.Write([]byte(":"))
		w.Write([]byte(strings.Join(v, ",")))
		w.Write([]byte("\n"))
	}
}

// From Amazon API documentation:
//
// CanonicalizedResource = [ "/" + Bucket ] +
//    <HTTP-Request-URI, from the protocol name up to the query string> +
//    [ subresource, if present. For example "?acl", "?location", "?logging", or "?torrent"];
func (b *Bucket) writeCanonicializedResource(w io.Writer, r *http.Request) {
	w.Write([]byte("/"))
	w.Write([]byte(b.Name))
	w.Write([]byte(r.URL.Path))
	b.writeSubResource(w, r)
}

// See Amazon S3 Developer Guide for explanation
// http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html
func (b *Bucket) writeSubResource(w io.Writer, r *http.Request) {
	var sr []string
	for k, vs := range r.URL.Query() {
		if paramsToSign[k] {
			for _, v := range vs {
				if v == "" {
					sr = append(sr, k)
				} else {
					sr = append(sr, k+"="+v)
				}
			}
		}
	}
	sort.Strings(sr)
	var q byte = '?'
	for _, s := range sr {
		w.Write([]byte{q})
		w.Write([]byte(s))
		q = '&'
	}
}
