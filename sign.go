package s3gof3r

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	prefix    = "AWS4-HMAC-SHA256"
	isoFormat = "20060102T150405Z"
	shortDate = "20060102"
)

var ignoredHeaders = map[string]bool{
	"Authorization":  true,
	"Content-Type":   true,
	"Content-Length": true,
	"User-Agent":     true,
}

type signer struct {
	Time    time.Time
	Request *http.Request
	Region  string
	Keys    Keys

	credentialString string
	signedHeaders    string
	signature        string

	canonicalHeaders string
	canonicalString  string
	stringToSign     string
}

// Sign signs the http.Request
func (b *Bucket) Sign(req *http.Request) {
	if req.Header == nil {
		req.Header = http.Header{}
	}
	if b.S3.Keys.SecurityToken != "" {
		req.Header.Set("X-Amz-Security-Token", b.S3.Keys.SecurityToken)
	}
	req.Header.Set("User-Agent", "S3Gof3r")
	s := &signer{
		Time:    time.Now(),
		Request: req,
		Region:  b.S3.Region(),
		Keys:    b.S3.Keys,
	}
	s.sign()
}

func (s *signer) sign() {
	s.buildTime()
	s.buildCredentialString()
	s.buildCanonicalHeaders()
	s.buildCanonicalString()
	s.buildStringToSign()
	s.buildSignature()
	parts := []string{
		prefix + " Credential=" + s.Keys.AccessKey + "/" + s.credentialString,
		"SignedHeaders=" + s.signedHeaders,
		"Signature=" + s.signature,
	}
	s.Request.Header.Set("Authorization", strings.Join(parts, ","))
}

func (s *signer) buildTime() {
	s.Request.Header.Set("X-Amz-Date", s.Time.UTC().Format(isoFormat))
}

func (s *signer) buildCredentialString() {
	s.credentialString = strings.Join([]string{
		s.Time.UTC().Format(shortDate),
		s.Region,
		"s3",
		"aws4_request",
	}, "/")
}

func (s *signer) buildCanonicalHeaders() {
	var headers []string
	headers = append(headers, "host")
	for k := range s.Request.Header {
		if _, ok := ignoredHeaders[http.CanonicalHeaderKey(k)]; ok {
			continue
		}
		headers = append(headers, strings.ToLower(k))
	}
	sort.Strings(headers)

	s.signedHeaders = strings.Join(headers, ";")

	headerValues := make([]string, len(headers))
	for i, k := range headers {
		if k == "host" {
			headerValues[i] = "host:" + s.Request.URL.Host
		} else {
			headerValues[i] = k + ":" +
				strings.Join(s.Request.Header[http.CanonicalHeaderKey(k)], ",")
		}
	}

	s.canonicalHeaders = strings.Join(headerValues, "\n")
}

func (s *signer) buildCanonicalString() {
	s.Request.URL.RawQuery = strings.Replace(s.Request.URL.Query().Encode(), "+", "%20", -1)
	uri := s.Request.URL.Opaque
	if uri != "" {
		uri = "/" + strings.Join(strings.Split(uri, "/")[3:], "/")
	} else {
		uri = s.Request.URL.EscapedPath()
	}
	if uri == "" {
		uri = "/"
	}

	s.canonicalString = strings.Join([]string{
		s.Request.Method,
		uri,
		s.Request.URL.RawQuery,
		s.canonicalHeaders + "\n",
		s.signedHeaders,
		s.bodyDigest(),
	}, "\n")
}

func (s *signer) buildStringToSign() {
	s.stringToSign = strings.Join([]string{
		prefix,
		s.Time.UTC().Format(isoFormat),
		s.credentialString,
		hex.EncodeToString(sha([]byte(s.canonicalString))),
	}, "\n")
}

func (s *signer) buildSignature() {
	secret := s.Keys.SecretKey
	date := hmacSign([]byte("AWS4"+secret), []byte(s.Time.UTC().Format(shortDate)))
	region := hmacSign(date, []byte(s.Region))
	service := hmacSign(region, []byte("s3"))
	credentials := hmacSign(service, []byte("aws4_request"))
	signature := hmacSign(credentials, []byte(s.stringToSign))
	s.signature = hex.EncodeToString(signature)
}

func (s *signer) bodyDigest() string {
	hash := s.Request.Header.Get("X-Amz-Content-Sha256")
	if hash == "" {
		if s.Request.Body == nil {
			hash = hex.EncodeToString(sha([]byte{}))
		} else {
			body, _ := ioutil.ReadAll(s.Request.Body)
			s.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
			hash = hex.EncodeToString(sha(body))
		}
		s.Request.Header.Add("X-Amz-Content-Sha256", hash)
	}
	return hash
}

func hmacSign(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func sha(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

func shaReader(r io.ReadSeeker) string {
	hash := sha256.New()
	start, _ := r.Seek(0, 1)
	defer r.Seek(start, 0)

	io.Copy(hash, r)
	sum := hash.Sum(nil)
	return hex.EncodeToString(sum)
}
