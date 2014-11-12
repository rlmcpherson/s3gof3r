package main

import (
	"log"
	"net/http"
	"os"
	"os/user"
	"reflect"
	"testing"
)

func TestHomeDir(t *testing.T) {
	hs := os.Getenv("HOME")
	defer os.Setenv("HOME", hs)

	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	thdir := u.HomeDir

	if err := os.Setenv("HOME", ""); err != nil {
		t.Fatal(err)
	}
	hdir, err := homeDir()
	if err != nil {
		t.Fatal(err)
	}
	if hdir != thdir {
		t.Errorf("expected %s\n actual%s\n", thdir, hdir)
	}

}

func TestACL(t *testing.T) {
	h2 := http.Header{"X-Amz-Acl": []string{"public-read"}}
	h3 := ACL(http.Header{}, "public-read")
	if !reflect.DeepEqual(h3, h2) {
		log.Fatalf("mismatch: %v, %v", h2, h3)
	}
}
