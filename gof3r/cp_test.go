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
var defaultcpOpts = &cpOpts{
	CommonOpts: CommonOpts{EndPoint: "s3.amazonaws.com"},
	DataOpts:   DataOpts{PartSize: mb}}

type cpTest struct {
	*cpOpts
	args []string
	err  error
}

var cpTests = []cpTest{
	{defaultcpOpts,
		[]string{"cp_test.go", "s3://" + tb + "/t1"},
		nil},
	{defaultcpOpts,
		[]string{"s3://" + tb + "/t1", "s3://" + tb + "/t2"},
		nil},
	{defaultcpOpts,
		[]string{"s3://" + tb + "/t1", "s3://" + tb + "//t2"},
		nil},
	{defaultcpOpts,
		[]string{"s3://" + tb + "/t1", "/dev/null"},
		nil},
	{defaultcpOpts,
		[]string{"s3://" + tb + "/noexist", "/dev/null"},
		errors.New("404")},
	{&cpOpts{
		CommonOpts: CommonOpts{EndPoint: "s3-external-1.amazonaws.com"},
		DataOpts:   DataOpts{PartSize: mb}},
		[]string{"s3://" + tb + "/&exist", "/dev/null"},
		errors.New("404")},
	{&cpOpts{
		DataOpts: DataOpts{NoSSL: true,
			PartSize: mb}},
		[]string{"s3://" + tb + "/t1", "s3://" + tb + "/tdir/.tst"},
		nil},
	{defaultcpOpts,
		[]string{"s3://" + tb + "/t1"},
		errors.New("source and destination arguments required")},
	{defaultcpOpts,
		[]string{"s://" + tb + "/t1", "s3://" + tb + "/tdir/.tst"},
		errors.New("parse error: s://")},
	{defaultcpOpts,
		[]string{"http://%%s", ""},
		errors.New("parse error: parse http")},
	{defaultcpOpts,
		[]string{"s3://" + tb + "/t1", "s3://no-bucket/.tst"},
		errors.New("bucket does not exist")},
}

func TestcpOptsExecute(t *testing.T) {

	if tb == "" {
		t.Fatal("TEST_BUCKET must be set in environment")
	}

	for _, tt := range cpTests {
		t.Log(tt)
		err := tt.Execute(tt.args)
		errComp(tt.err, err, t, tt)
	}

}
