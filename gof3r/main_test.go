package main

import (
	"errors"

	"os"
	"strings"
	"testing"
)

type flagTest struct {
	flags []string
	err   error
}

var flagTests = []flagTest{

	{[]string{"gof3r", "nocmd"},
		errors.New("Unknown command")},
	{[]string{"gof3r", "put", "-b", "fake-bucket", "-k", "test-key"},
		errors.New("Access Denied")},
	{[]string{"gof3r", "put", "-b", "fake-bucket", "-k", "key",
		"-c", "1", "-s", "1024", "--debug", "--no-ssl", "--no-md5"},
		errors.New("Access Denied")},
	{[]string{"gof3r", "get", "-b", "fake-bucket", "-k", "test-key"},
		errors.New("Access Denied")},
	{[]string{"gof3r", "get", "-b", "fake-bucket", "-k", "key",
		"-c", "1", "-s", "1024", "--debug", "--no-ssl", "--no-md5"},
		errors.New("Access Denied")},
	{[]string{"gof3r", "put"},
		errors.New("required flags")},
	{[]string{"gof3r", "put", "-b"},
		errors.New("expected argument for flag")},
	{[]string{"gof3r", "get", "-b"},
		errors.New("expected argument for flag")},
	{[]string{"gof3r", "get"},
		errors.New("required flags")},
}

func TestFlags(t *testing.T) {
	for _, tt := range flagTests {
		os.Args = tt.flags
		_, err := parser.Parse()
		errComp(tt.err, err, t, tt)
	}
}

func errComp(expect, actual error, t *testing.T, tt interface{}) bool {

	if expect == nil && actual == nil {
		return true
	}

	if expect == nil || actual == nil {
		t.Errorf("gof3r called with %v\n Expected: %v\n Actual:   %v\n", tt, expect, actual)
		return false
	}
	if !strings.Contains(actual.Error(), expect.Error()) {
		t.Errorf("gof3r called with %v\n Expected: %v\n Actual:   %v\n", tt, expect, actual)
		return false
	}
	return true

}
