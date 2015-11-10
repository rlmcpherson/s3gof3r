package s3gof3r

import (
	"net/http"
	"testing"
	"time"
)

func newSigner(region string) *signer {
	endpoint := "https://examplebucket.s3.amazonaws.com"
	req, _ := http.NewRequest("GET", endpoint, nil)
	req.URL.Path = "/test.txt"
	req.Header.Add("Range", "bytes=0-9")
	return &signer{
		Request: req,
		Time:    time.Date(2013, 05, 24, 0, 0, 0, 0, time.UTC),
		Region:  region,
		Keys: Keys{
			AccessKey: "AKIAIOSFODNN7EXAMPLE",
			SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
	}
}

func TestBuildTime(t *testing.T) {
	s := newSigner("us-east-1")
	s.buildTime()
	expect := "20130524T000000Z"
	if date := s.Request.Header.Get("X-Amz-Date"); date != expect {
		t.Errorf("date don't match, got '%s', expected '%s'", date, expect)
	}
}

func TestBuildCredentialString(t *testing.T) {
	s := newSigner("us-east-1")
	s.bodyDigest()
	s.buildCredentialString()
	expect := "20130524/us-east-1/s3/aws4_request"
	if s.credentialString != expect {
		t.Errorf("credential string don't match, got '%s', expected '%s'",
			s.credentialString, expect)
	}
}

func TestBuildCanonicalHeaders(t *testing.T) {
	s := newSigner("us-east-1")
	s.buildTime()
	s.buildCredentialString()
	s.buildCanonicalHeaders()

	expectSigned := "host;range;x-amz-date"
	if s.signedHeaders != expectSigned {
		t.Errorf("signed headers don't match, got '%s', expected '%s'",
			s.signedHeaders, expectSigned)
	}

	expectCanonical := `host:examplebucket.s3.amazonaws.com
range:bytes=0-9
x-amz-date:20130524T000000Z`
	if s.canonicalHeaders != expectCanonical {
		t.Errorf("canonical headers don't match, got '%s', expected '%s'",
			s.canonicalHeaders, expectCanonical)
	}
}

func TestBuildCanonicalString(t *testing.T) {
	s := newSigner("us-east-1")
	s.buildTime()
	s.buildCredentialString()
	s.buildCanonicalHeaders()
	s.buildCanonicalString()
	expect := `GET
/test.txt

host:examplebucket.s3.amazonaws.com
range:bytes=0-9
x-amz-date:20130524T000000Z

host;range;x-amz-date
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
	if s.canonicalString != expect {
		t.Errorf("canonical string don't match, got '%s', expected '%s'",
			s.canonicalString, expect)
	}
}

func TestBuildStringToSign(t *testing.T) {
	s := newSigner("us-east-1")
	s.buildTime()
	s.buildCredentialString()
	s.buildCanonicalHeaders()
	s.buildCanonicalString()
	s.buildStringToSign()
	expect := `AWS4-HMAC-SHA256
20130524T000000Z
20130524/us-east-1/s3/aws4_request
8946e8df7a95b4714c63ae8664bbab443f99610b0858e8966eac22c72dae0232`
	if s.stringToSign != expect {
		t.Errorf("string to sign don't match, got '%s', expected '%s'",
			s.stringToSign, expect)
	}
}

func TestBuildSignature(t *testing.T) {
	s := newSigner("us-east-1")
	s.buildTime()
	s.buildCredentialString()
	s.buildCanonicalHeaders()
	s.buildCanonicalString()
	s.buildStringToSign()
	s.buildSignature()
	expect := "b4904babad39b29ebe2eaefecf4c7037be9c6362be0aebe68ea5c700020e5085"
	if s.signature != expect {
		t.Errorf("signature don't match, got '%s', expected '%s'",
			s.signature, expect)
	}
}
