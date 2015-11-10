package s3gof3r

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"math"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// defined by amazon
const (
	minPartSize  = 5 * mb
	maxPartSize  = 5 * gb
	maxObjSize   = 5 * tb
	maxNPart     = 10000
	md5Header    = "content-md5"
	sha256Header = "X-Amz-Content-Sha256"
)

type part struct {
	r   io.ReadSeeker
	len int64
	b   []byte

	// Read by xml encoder
	PartNumber int
	ETag       string

	// Checksums
	md5    string
	sha256 string
}

type putter struct {
	url url.URL
	b   *Bucket
	c   *Config

	bufsz      int64
	buf        []byte
	bufbytes   int // bytes written to current buffer
	ch         chan *part
	part       int
	closed     bool
	err        error
	wg         sync.WaitGroup
	md5OfParts hash.Hash
	md5        hash.Hash
	ETag       string

	sp *bp

	makes    int
	UploadID string `xml:"UploadId"`
	xml      struct {
		XMLName string `xml:"CompleteMultipartUpload"`
		Part    []*part
	}
	putsz int64
}

// Sends an S3 multipart upload initiation request.
// See http://docs.amazonwebservices.com/AmazonS3/latest/dev/mpuoverview.html.
// The initial request returns an UploadId that we use to identify
// subsequent PUT requests.
func newPutter(url url.URL, h http.Header, c *Config, b *Bucket) (p *putter, err error) {
	p = new(putter)
	p.url = url
	p.c, p.b = new(Config), new(Bucket)
	*p.c, *p.b = *c, *b
	p.c.Concurrency = max(c.Concurrency, 1)
	p.c.NTry = max(c.NTry, 1)
	p.bufsz = max64(minPartSize, c.PartSize)
	resp, err := p.retryRequest("POST", url.String()+"?uploads", nil, h)
	if err != nil {
		return nil, err
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 200 {
		return nil, newRespError(resp)
	}
	err = xml.NewDecoder(resp.Body).Decode(p)
	if err != nil {
		return nil, err
	}
	p.ch = make(chan *part)
	for i := 0; i < p.c.Concurrency; i++ {
		go p.worker()
	}
	p.md5OfParts = md5.New()
	p.md5 = md5.New()

	p.sp = bufferPool(p.bufsz)

	return p, nil
}

func (p *putter) Write(b []byte) (int, error) {
	if p.closed {
		p.abort()
		return 0, syscall.EINVAL
	}
	if p.err != nil {
		p.abort()
		return 0, p.err
	}
	nw := 0
	for nw < len(b) {
		if p.buf == nil {
			p.buf = <-p.sp.get
			if int64(cap(p.buf)) < p.bufsz {
				p.buf = make([]byte, p.bufsz)
				runtime.GC()
			}
		}
		n := copy(p.buf[p.bufbytes:], b[nw:])
		p.bufbytes += n
		nw += n

		if len(p.buf) == p.bufbytes {
			p.flush()
		}
	}
	return nw, nil
}

func (p *putter) flush() {
	p.wg.Add(1)
	p.part++
	p.putsz += int64(p.bufbytes)
	part := &part{
		r:          bytes.NewReader(p.buf[:p.bufbytes]),
		len:        int64(p.bufbytes),
		b:          p.buf,
		PartNumber: p.part,
	}
	var err error
	part.md5, part.sha256, part.ETag, err = p.hashContent(part.r)
	if err != nil {
		p.err = err
	}

	p.xml.Part = append(p.xml.Part, part)
	p.ch <- part
	p.buf, p.bufbytes = nil, 0

	// if necessary, double buffer size every 2000 parts due to the 10000-part AWS limit
	// to reach the 5 Terabyte max object size, initial part size must be ~85 MB
	if p.part%2000 == 0 && p.part < maxNPart && growPartSize(p.part, p.bufsz, p.putsz) {
		p.bufsz = min64(p.bufsz*2, maxPartSize)
		p.sp.sizech <- p.bufsz // update pool buffer size
		logger.debugPrintf("part size doubled to %d", p.bufsz)
	}
}

func (p *putter) worker() {
	for part := range p.ch {
		p.retryPutPart(part)
	}
}

// Calls putPart up to nTry times to recover from transient errors.
func (p *putter) retryPutPart(part *part) {
	defer p.wg.Done()
	var err error
	for i := 0; i < p.c.NTry; i++ {
		err = p.putPart(part)
		if err == nil {
			p.sp.give <- part.b
			part.b = nil
			return
		}
		logger.debugPrintf("Error on attempt %d: Retrying part: %d, Error: %s", i, part.PartNumber, err)
		time.Sleep(time.Duration(math.Exp2(float64(i))) * 100 * time.Millisecond) // exponential back-off
	}
	p.err = err
}

// uploads a part, checking the etag against the calculated value
func (p *putter) putPart(part *part) error {
	v := url.Values{}
	v.Set("partNumber", strconv.Itoa(part.PartNumber))
	v.Set("uploadId", p.UploadID)
	if _, err := part.r.Seek(0, 0); err != nil { // move back to beginning, if retrying
		return err
	}
	req, err := http.NewRequest("PUT", p.url.String()+"?"+v.Encode(), part.r)
	if err != nil {
		return err
	}
	req.ContentLength = part.len
	req.Header.Set(md5Header, part.md5)
	req.Header.Set(sha256Header, part.sha256)
	p.b.Sign(req)
	resp, err := p.c.Client.Do(req)
	if err != nil {
		return err
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 200 {
		return newRespError(resp)
	}
	s := resp.Header.Get("etag")
	if len(s) < 2 {
		return fmt.Errorf("Got Bad etag:%s", s)
	}
	s = s[1 : len(s)-1] // includes quote chars for some reason
	if part.ETag != s {
		return fmt.Errorf("Response etag does not match. Remote:%s Calculated:%s", s, p.ETag)
	}
	return nil
}

func (p *putter) Close() (err error) {
	if p.closed {
		p.abort()
		return syscall.EINVAL
	}
	if p.err != nil {
		p.abort()
		return p.err
	}
	if p.bufbytes > 0 || // partial part
		p.part == 0 { // 0 length file
		p.flush()
	}
	p.wg.Wait()
	close(p.ch)
	p.closed = true
	close(p.sp.quit)

	// check p.err before completing
	if p.err != nil {
		p.abort()
		return p.err
	}
	// Complete Multipart upload
	body, err := xml.Marshal(p.xml)
	if err != nil {
		p.abort()
		return
	}
	b := bytes.NewReader(body)
	v := url.Values{}
	v.Set("uploadId", p.UploadID)
	resp, err := p.retryRequest("POST", p.url.String()+"?"+v.Encode(), b, nil)
	if err != nil {
		p.abort()
		return
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 200 {
		p.abort()
		return newRespError(resp)
	}
	// Check md5 hash of concatenated part md5 hashes against ETag
	// more info: https://forums.aws.amazon.com/thread.jspa?messageID=456442&#456442
	calculatedMd5ofParts := fmt.Sprintf("%x", p.md5OfParts.Sum(nil))
	// Parse etag from body of response
	err = xml.NewDecoder(resp.Body).Decode(p)
	if err != nil {
		return
	}
	// strip part count from end and '"' from front.
	remoteMd5ofParts := strings.Split(p.ETag, "-")[0]
	if len(remoteMd5ofParts) == 0 {
		return fmt.Errorf("Nil ETag")
	}
	remoteMd5ofParts = remoteMd5ofParts[1:len(remoteMd5ofParts)]
	if calculatedMd5ofParts != remoteMd5ofParts {
		if err != nil {
			return err
		}
		return fmt.Errorf("MD5 hash of part hashes comparison failed. Hash from multipart complete header: %s."+
			" Calculated multipart hash: %s.", remoteMd5ofParts, calculatedMd5ofParts)
	}
	if p.c.Md5Check {
		for i := 0; i < p.c.NTry; i++ {
			if err = p.putMd5(); err == nil {
				break
			}
		}
	}
	return
}

// Try to abort multipart upload. Do not error on failure.
func (p *putter) abort() {
	v := url.Values{}
	v.Set("uploadId", p.UploadID)
	s := p.url.String() + "?" + v.Encode()
	resp, err := p.retryRequest("DELETE", s, nil, nil)
	if err != nil {
		logger.Printf("Error aborting multipart upload: %v\n", err)
		return
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 204 {
		logger.Printf("Error aborting multipart upload: %v", newRespError(resp))
	}
	return
}

// Md5 functions
func (p *putter) hashContent(r io.ReadSeeker) (string, string, string, error) {
	m := md5.New()
	s := sha256.New()
	mw := io.MultiWriter(m, s, p.md5)
	if _, err := io.Copy(mw, r); err != nil {
		return "", "", "", err
	}
	md5Sum := m.Sum(nil)
	shaSum := hex.EncodeToString(s.Sum(nil))
	etag := hex.EncodeToString(md5Sum)
	// add to checksum of all parts for verification on upload completion
	if _, err := p.md5OfParts.Write(md5Sum); err != nil {
		return "", "", "", err
	}
	return base64.StdEncoding.EncodeToString(md5Sum), shaSum, etag, nil
}

// Put md5 file in .md5 subdirectory of bucket  where the file is stored
// e.g. the md5 for https://mybucket.s3.amazonaws.com/gof3r will be stored in
// https://mybucket.s3.amazonaws.com/.md5/gof3r.md5
func (p *putter) putMd5() (err error) {
	calcMd5 := fmt.Sprintf("%x", p.md5.Sum(nil))
	md5Reader := strings.NewReader(calcMd5)
	md5Path := fmt.Sprint(".md5", p.url.Path, ".md5")
	md5Url, err := p.b.url(md5Path, p.c)
	if err != nil {
		return err
	}
	logger.debugPrintln("md5: ", calcMd5)
	logger.debugPrintln("md5Path: ", md5Path)
	r, err := http.NewRequest("PUT", md5Url.String(), md5Reader)
	if err != nil {
		return
	}
	p.b.Sign(r)
	resp, err := p.c.Client.Do(r)
	if err != nil {
		return
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 200 {
		return newRespError(resp)
	}
	return
}

func (p *putter) retryRequest(method, urlStr string, body io.ReadSeeker, h http.Header) (resp *http.Response, err error) {
	for i := 0; i < p.c.NTry; i++ {
		var req *http.Request
		req, err = http.NewRequest(method, urlStr, body)
		if err != nil {
			return
		}
		for k := range h {
			for _, v := range h[k] {
				req.Header.Add(k, v)
			}
		}

		if body != nil {
			req.Header.Set(sha256Header, shaReader(body))
		}

		p.b.Sign(req)
		resp, err = p.c.Client.Do(req)
		if err == nil {
			return
		}
		logger.debugPrintln(err)
		if body != nil {
			if _, err = body.Seek(0, 0); err != nil {
				return
			}
		}
	}
	return
}

// returns true unless partSize is large enough
// to achieve maxObjSize with remaining parts
func growPartSize(partIndex int, partSize, putsz int64) bool {
	return (maxObjSize-putsz)/(maxNPart-int64(partIndex)) > partSize
}
