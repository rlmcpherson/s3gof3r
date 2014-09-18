package main

import (
	"os"
	"os/user"
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
