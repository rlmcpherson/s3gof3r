Package s3gof3r provides fast, concurrent, streaming access to Amazon S3. Includes a command-line interface.

 Command-line Interface Usage:

 ```
   To stream up to S3:
      $  <input_stream> | gof3r put -b <bucket> -k <s3_path>
   To stream down from S3:
      $ gof3r get -b <bucket> -k <s3_path> | <output_stream>
   To upload a file to S3:
      $ gof3r  put --path=<local_path> --bucket=<bucket> --key=<s3_path> --header=<http_header1> --header=<http_header2>...
   To download a file from S3:
      $ gof3r  get --bucket=<bucket> --key=<s3_path>
```

 Set AWS keys as environment Variables (required):

```
  $ export AWS_ACCESS_KEY_ID=<access_key>
  $ export AWS_SECRET_ACCESS_KEY=<secret_key>
```

 Examples:

  ```
  $ tar -cf - /foo_dir/ | gof3r put -b my_s3_bucket -k bar_dir/s3_object -m x-amz-meta-custom-metadata:abc123 -m x-amz-server-side-encryption:AES256
  $ gof3r get -b my_s3_bucket -k bar_dir/s3_object | tar -x
  ```
 Complete Usage: get command:

 ```
   gof3r [OPTIONS] get [get-OPTIONS]

   get (download) from S3

   Help Options:
   -h, --help          Show this help message

   get (download) from S3:
   -p, --path=         Path to file. Defaults to standard output for streaming. (/dev/stdout)
   -k, --key=          key of s3 object
   -b, --bucket=       s3 bucket
   --md5Check-off      Do not use md5 hash checking to ensure data integrity.
		       By default, the md5 hash of is calculated concurrently
		       during puts, stored at <bucket>.md5/<key>.md5, and verified on gets.
   -c, --concurrency=  Concurrency of transfers (20)
   -s, --partsize=     initial size of concurrent parts, in bytes (20 MB)
   --debug             Print debug statements and dump stacks.

   Help Options:
   -h, --help          Show this help message
```

 Complete Usage: put command:

 ```
   gof3r [OPTIONS] put [put-OPTIONS]

   put (upload)to S3

   Help Options:
     -h, --help          Show this help message

   put (upload) to S3:
     -p, --path=         Path to file. Defaults to standard input for streaming. (/dev/stdin)
     -m, --header=       HTTP headers
     -k, --key=          key of s3 object
     -b, --bucket=       s3 bucket
	 --md5Check-off  Do not use md5 hash checking to ensure data integrity. By default, the md5 hash of is calculated concurrently
			 during puts, stored at <bucket>.md5/<key>.md5, and verified on gets.
     -c, --concurrency=  Concurrency of transfers (20)
     -s, --partsize=     initial size of concurrent parts, in bytes (20 MB)
	 --debug         Print debug statements and dump stacks.

   Help Options:
     -h, --help          Show this help message
 ```

See godoc.org for more documentation:

package: [http://godoc.org/github.com/rlmcpherson/s3gof3r](http://godoc.org/github.com/rlmcpherson/s3gof3r)
command-line interface: [http://godoc.org/github.com/rlmcpherson/s3gof3r/gof3r](http://godoc.org/github.com/rlmcpherson/s3gof3r/gof3r)
