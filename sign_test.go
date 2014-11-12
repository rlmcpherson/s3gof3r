package s3gof3r

import (
	"testing"
	"time"
)

var s3 *S3

//Testing http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html
func init() {
	keys := Keys{
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	s3 = New(DefaultDomain, keys)
}

func TestPresign(t *testing.T) {
	bucket := s3.Bucket("johnsmith")

	uri, err := bucket.Presign(
		"GET",
		"http://johnsmith.s3.amazonaws.com/photos/puppy.jpg",
		time.Unix(1175139620, 0),
	)

	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	if uri.Host != "johnsmith.s3.amazonaws.com" {
		t.Error("Invalid Host:", uri.Host)
	}

	if uri.Path != "/photos/puppy.jpg" {
		t.Error("Invalid Path:", uri.Path)
	}

	signature := uri.Query().Get("Signature")
	if signature != "NpgCjnDzrM+WFzoENXmpNDUsSn8=" {
		t.Error("Invalid Signature:", signature)
	}

	expiry := uri.Query().Get("Expires")
	if expiry != "1175139620" {
		t.Error("Invalid Expiry:", expiry)
	}
}
