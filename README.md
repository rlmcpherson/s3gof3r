# s3gof3r  [![Build Status](https://travis-ci.org/rlmcpherson/s3gof3r.svg?branch=master)](https://travis-ci.org/rlmcpherson/s3gof3r) [![GoDoc](https://godoc.org/github.com/rlmcpherson/s3gof3r?status.png)](https://godoc.org/github.com/rlmcpherson/s3gof3r)

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

- *Memory Efficiency:* Memory used to upload and download parts is recycled. For an upload or download with the default concurrency of 10 and part size of 20 MB, the maximum memory usage is less than 300 MB. Memory footprint can be further reduced by reducing part size or concurrency. 



## Installation

s3gof3r is written in Go and requires go 1.5 or later. It can be installed with `go get` to download and compile it from source. To install the command-line tool, `gof3r` set `GO15VENDOREXPERIMENT=1` in your environment:

    $ go get github.com/rlmcpherson/s3gof3r/gof3r
    
To install just the package for use in other Go programs:

    $ go get github.com/rlmcpherson/s3gof3r

### Release Binaries

To try the latest release of the gof3r command-line interface without installing go, download the statically-linked binary for your architecture from **[Github Releases](https://github.com/rlmcpherson/s3gof3r/releases).**



## gof3r (command-line interface) usage:

 ```
   To stream up to S3:
      $  <input_stream> | gof3r put -b <bucket> -k <s3_path>
   To stream down from S3:
      $ gof3r get -b <bucket> -k <s3_path> | <output_stream>
   To upload a file to S3:
      $ $ gof3r cp <local_path> s3://<bucket>/<s3_path>
   To download a file from S3:
      $ gof3r cp s3://<bucket>/<s3_path> <local_path>
```

 Set AWS keys as environment Variables:

```
  $ export AWS_ACCESS_KEY_ID=<access_key>
  $ export AWS_SECRET_ACCESS_KEY=<secret_key>
```

gof3r also supports [IAM role](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html)-based keys from EC2 instance metadata. If available and environment variables are not set, these keys are used are used automatically.

 Examples:

  ```
  $ tar -cf - /foo_dir/ | gof3r put -b my_s3_bucket -k bar_dir/s3_object -m x-amz-meta-custom-metadata:abc123 -m x-amz-server-side-encryption:AES256
  $ gof3r get -b my_s3_bucket -k bar_dir/s3_object | tar -x    
  ```  
  **see the [gof3r man page](http://randallmcpherson.com/gof3r.html) for complete usage**
 
## Documentation

**s3gof3r package:** See the [godocs](http://godoc.org/github.com/rlmcpherson/s3gof3r) for api documentation.

**gof3r cli :**  [godoc](http://godoc.org/github.com/rlmcpherson/s3gof3r/gof3r) and [gof3r man page](http://randallmcpherson.com/gof3r.html)


Have a question? Ask it on the [s3gof3r Mailing List](https://groups.google.com/forum/#!forum/s3gof3r)
