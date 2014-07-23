package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

// CommonOpts are Options common to all commands
type CommonOpts struct {
	NoSSL       bool   `long:"no-ssl" description:"Do not use SSL for endpoint connection."`
	NoMd5       bool   `long:"no-md5" description:"Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently during puts, stored at <bucket>.md5/<key>.md5, and verified on gets."`
	Concurrency int    `long:"concurrency" short:"c" default:"10" description:"Concurrency of transfers"`
	PartSize    int64  `long:"partsize" short:"s" description:"Initial size of concurrent parts, in bytes" default:"20971520"`
	EndPoint    string `long:"endpoint" description:"Amazon S3 endpoint" default:"s3.amazonaws.com"`
	Debug       bool   `long:"debug" description:"Enable debug logging."`
}

var AppOpts struct {
	Version func() `long:"version" short:"v" description:"Print version"`
	Man     func() `long:"manpage" short:"m" description:"Create gof3r.man man page in current directory"`
}

var parser = flags.NewParser(&AppOpts, (flags.HelpFlag | flags.PassDoubleDash))

func init() {

	// set parser fields
	parser.ShortDescription = "streaming, concurrent s3 client"

	AppOpts.Version = func() {
		fmt.Fprintf(os.Stderr, "%s version %s\n", name, version)
		os.Exit(0)
	}

	AppOpts.Man = func() {
		f, err := os.Create(name + ".man")
		if err != nil {
			log.Fatal(err)
		}
		parser.WriteManPage(f)
		fmt.Fprintf(os.Stderr, "man page written to %s\n", f.Name())
		os.Exit(0)
	}
}
