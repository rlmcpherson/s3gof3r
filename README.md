# s3gof3r #

s3gof3r provides fast, parallelized, pipelined streaming access to Amazon S3. It includes a command-line interface: `gof3r`.

It is optimized for high speed transfer of large objects into and out of Amazon S3. Streaming support allows for usage like:

```
  $ tar -czf - <my_dir/> | gof3r put -b <s3_bucket> -k <s3_object>    
  $ gof3r get -b <s3_bucket> -k <s3_object> | tar -zx
```


**Speed Benchmarks**

On an EC2 instance, gof3r can exceed 1 Gbps for both puts and gets:

```
  $ gof3r get -b test-bucket -k 8_GB_tar | pv -a | tar -x
  Duration: 53.201632211s
  [ 167MB/s]
  

  $ tar -cf - test_dir/ | pv -a | gof3r put -b test-bucket -k 8_GB_tar
  Duration: 1m16.080800315s
  [ 119MB/s]
```

These tests were performed on an m1.xlarge EC2 instance with a virtualized 1 Gigabit ethernet interface. See [Amazon EC2 Instance Details](http://aws.amazon.com/ec2/instance-types/instance-details/) for more information.


**Features**

- *Speed:* Especially for larger s3 objects where parallelism can be exploited, s3gof3r will saturate the bandwidth of an EC2 instance. See the Benchmarks above.

- *Streaming Uploads and Downloads:* As the above examples illustrate, streaming allows the gof3r command-line tool to be used with linux/unix pipes. This allows transformation of the data in parallel as it is uploaded or downloaded from S3.

- *End-to-end Integrity Checking:* s3gof3r calculates the md5 hash of the stream in parallel while uploading and downloading. On upload, a file containing the md5 hash is saved in s3. This is checked against the calculated md5 on download. On upload, the content-md5 of each part is calculated and sent with the header to be checked by AWS. s3gof3r also checks the 'hash of hashes' returned by S3 in the `Etag` field on completion of a multipart upload. See the [S3 API Reference](http://docs.aws.amazon.com/AmazonS3/latest/API/mpUploadComplete.html) for details.

- *Retry Everything:* All http requests and every part is retried on both uploads and downloads. Requests to S3 frequently time out, especially under high load, so this is essential to complete large uploads or downloads.

- *Memory Efficiency:* Memory used to upload and download parts is recycled. For an upload with the default concurrency of 10 and part size of 20 MB, the maximum memory usage is less than 250 MB and does not depend on the size of the upload. For downloads with the same default configuration, maximum memory usage will not exceed 450 MB. The additional memory usage vs. uploading is due to the need to reorder parts before adding them to the stream.




## Installation ##

s3gof3r is written in Go and requires a Go installation. It can be installed with `go get` to download and compile it from source. To install the command-line tool, `gof3r`:

    $ go get github.com/rlmcpherson/s3gof3r/gof3r
    
To install just the package for use in other Go programs:

    $ go get github.com/rlmcpherson/s3gof3r

### Release Binaries ###

To try the latest release of the gof3r command-line interface without installing go, download the statically-linked binary for your architecture from **[Github Releases](https://github.com/rlmcpherson/s3gof3r/releases).**



## Gof3r (Command-line Interface) Usage: ##

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

 Set AWS keys as environment Variables:

```
  $ export AWS_ACCESS_KEY_ID=<access_key>
  $ export AWS_SECRET_ACCESS_KEY=<secret_key>
```

Gof3r also supports IAM role-based keys from EC2 instance metadata. If available and environment variables are not set, these keys are used are used automatically. See http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html.

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
 
## Documentation ##

**See godoc.org for full documentation, including the s3gof3r package api:**

s3gof3r package: [http://godoc.org/github.com/rlmcpherson/s3gof3r](http://godoc.org/github.com/rlmcpherson/s3gof3r)

command-line interface: [http://godoc.org/github.com/rlmcpherson/s3gof3r/gof3r](http://godoc.org/github.com/rlmcpherson/s3gof3r/gof3r)

## Mailing List ##

https://groups.google.com/forum/#!forum/s3gof3r
