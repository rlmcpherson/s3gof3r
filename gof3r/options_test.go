package main

import (
	"log"
	"net/http"
	"reflect"
	"testing"
)

func TestACL(t *testing.T) {
	h2 := http.Header{"X-Amz-Acl": []string{"public-read"}}
	h3 := ACL(http.Header{}, "public-read")
	if !reflect.DeepEqual(h3, h2) {
		log.Fatalf("mismatch: %v, %v", h2, h3)
	}
}
