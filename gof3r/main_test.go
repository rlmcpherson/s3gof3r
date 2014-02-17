package main

import (
	"testing"
)

type cmdTest struct {
	in       string
	expected string
}

var cmdTests = []cmdTest{
	{"get", "foo"},
	{"put", "foo"},
}

func TestMain(t *testing.T) {
	for _, tt := range cmdTests {
		actual := ""
		if actual != tt.expected {
			t.Errorf("gof3r called with %s. Expected %s, actual %s",
				tt.in, tt.expected, actual)

		}
	}
}
