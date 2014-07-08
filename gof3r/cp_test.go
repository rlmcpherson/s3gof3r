package main

import (
	"errors"
	"os"
	"testing"
)

// convenience multipliers
const (
	_        = iota
	kb int64 = 1 << (10 * iota)
	mb
	gb
)

var tb = os.Getenv("TEST_BUCKET")
var defaultCp = &Cp{EndPoint: "s3.amazonaws.com",
	PartSize: mb}

type cpTest struct {
	*Cp
	args []string
	err  error
}

var cpTests = []cpTest{
	{defaultCp,
		[]string{"cp_test.go", "s3://" + tb + "/t1"},
		nil},
	{defaultCp,
		[]string{"s3://" + tb + "/t1", "s3://" + tb + "/t2"},
		nil},
	{defaultCp,
		[]string{"s3://" + tb + "/t1", "s3://" + tb + "//t2"},
		nil},
	{defaultCp,
		[]string{"s3://" + tb + "/t1", "/dev/null"},
		nil},
	{defaultCp,
		[]string{"s3://" + tb + "/noexist", "/dev/null"},
		errors.New("404")},
	{&Cp{EndPoint: "s3-external-1.amazonaws.com",
		PartSize: mb},
		[]string{"s3://" + tb + "/&exist", "/dev/null"},
		errors.New("404")},
	{&Cp{NoSSL: true,
		PartSize: mb},
		[]string{"s3://" + tb + "/t1", "s3://" + tb + "/tdir/.tst"},
		nil},
	{&Cp{EndPoint: "s3.amazonaws.com",
		PartSize: mb},
		[]string{"s3://" + tb + "/t1"},
		errors.New("source and destination arguments required")},
	{&Cp{EndPoint: "s3.amazonaws.com",
		PartSize: mb},
		[]string{"s://" + tb + "/t1", "s3://" + tb + "/tdir/.tst"},
		errors.New("parse error: s://")},
	{defaultCp,
		[]string{"http://%%s", ""},
		errors.New("parse error: parse http")},
	{defaultCp,
		[]string{"s3://" + tb + "/t1", "s3://no-bucket/.tst"},
		errors.New("bucket does not exist")},
}

func TestCpExecute(t *testing.T) {

	if tb == "" {
		t.Fatal("TEST_BUCKET must be set in environment")
	}

	for _, tt := range cpTests {
		t.Log(tt)
		err := tt.Execute(tt.args)
		errComp(tt.err, err, t, tt)
	}

}
