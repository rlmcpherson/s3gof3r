package s3gof3r

import (
	"bytes"
	"container/list"
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// defined by amazon
const (
	minPartSize = 5 * mb
	maxPartSize = 5 * gb // for 32-bit use; amz max is 5GiB
	maxObjSize  = 5 * tb
	maxNPart    = 10000
	md5Header   = "content-md5"
)

type part struct {
	r   io.ReadSeeker
	len int64
	b   *bytes.Buffer

	// read by xml encoder
	PartNumber int
	ETag       string

	// Used for checksum of checksums on completion
	contentMd5 string
}

type putter struct {
	url         url.URL
	client      *http.Client
	b           *Bucket
	concurrency int
	nTry        int
	UploadId    string // written by xml decoder

	bufsz      int64
	buf        *bytes.Buffer
	ch         chan *part
	part       int
	closed     bool
	err        error
	wg         sync.WaitGroup
	md5OfParts hash.Hash
	ETag       string

	get   chan *bytes.Buffer
	give  chan *bytes.Buffer
	makes int

	xml struct {
		XMLName string `xml:"CompleteMultipartUpload"`
		Part    []*part
	}
}

type completeXml struct {
	ETag string
}

// Sends an S3 multipart upload initiation request.
// See http://docs.amazonwebservices.com/AmazonS3/latest/dev/mpuoverview.html.
// This initial request returns an UploadId that we use to identify
// subsequent PUT requests.
func newPutter(url url.URL, h http.Header, c *Config, b *Bucket) (p *putter, err error) {
	p = new(putter)
	p.url = url
	p.client = c.Client
	p.b = b
	p.concurrency = c.Concurrency
	p.nTry = c.NTry

	p.bufsz = max64(minPartSize, c.PartSize)
	r, err := http.NewRequest("POST", url.String()+"?uploads", nil)
	if err != nil {
		return nil, err
	}
	for k := range h {
		for _, v := range h[k] {
			r.Header.Add(k, v)
		}
	}
	p.b.Sign(r)
	resp, err := p.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, newRespError(resp)
	}
	err = xml.NewDecoder(resp.Body).Decode(p)
	if err != nil {
		return nil, err
	}
	p.ch = make(chan *part)
	for i := 0; i < p.concurrency; i++ {
		go p.worker()
	}
	p.md5OfParts = md5.New()
	//p.get, p.give = p.makeRecycler()
	p.get, p.give = startBufferPool(p.concurrency * 2)
	return p, nil
}

func (p *putter) Write(b []byte) (int, error) {
	if p.closed {
		return 0, syscall.EINVAL
	}
	if p.err != nil {
		return 0, p.err
	}
	if p.buf == nil {
		p.buf = p.get_buffer()
		p.buf.Reset()
	}
	n, err := p.buf.Write(b)
	if err != nil {
		return n, err
	}
	if int64(p.buf.Len()) >= (p.bufsz - int64(len(b))) {
		p.flush()
	}
	return n, nil
}

func (p *putter) flush() {
	p.wg.Add(1)
	p.part++
	b := *p.buf
	part := &part{bytes.NewReader(b.Bytes()), int64(b.Len()), p.buf, p.part, "", ""}
	var err error
	part.contentMd5, part.ETag, err = md5Content(part.r, p)
	if err != nil {
		p.err = err
	}

	p.xml.Part = append(p.xml.Part, part)
	p.ch <- part
	p.buf = nil
	// double buffer size every 500 parts to
	// avoid exceeding the 10000-part AWS limit
	// while still reaching the 5 Terabyte max object size
	if p.part%1000 == 0 {
		p.bufsz = min64(p.bufsz*2, maxPartSize)
	}

}

func (p *putter) worker() {
	for part := range p.ch {
		p.retryUploadPart(part)
	}
}

// Calls putPart up to nTry times to recover from transient errors.
func (p *putter) retryUploadPart(part *part) {
	defer p.wg.Done()
	var err error
	for i := 0; i < p.nTry; i++ {
		part.r.Seek(0, 0)
		err = p.putPart(part)
		if err == nil {
			p.give <- part.b
			return
		}
		log.Print(err)
	}
	p.err = err
}

// uploads a part, checking the etag against the calculated value
func (p *putter) putPart(part *part) error {
	v := url.Values{}
	v.Set("partNumber", strconv.Itoa(part.PartNumber))
	v.Set("uploadId", p.UploadId)
	req, err := http.NewRequest("PUT", p.url.String()+"?"+v.Encode(), part.r)
	if err != nil {
		return err
	}
	req.ContentLength = part.len
	req.Header.Set(md5Header, part.contentMd5)
	p.b.Sign(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return newRespError(resp)
	}
	s := resp.Header.Get("etag")
	s = s[1 : len(s)-1] // includes quote chars for some reason
	if part.ETag != s {
		return fmt.Errorf("Response etag does not match. Remote:%s Calculated:%s", s, p.ETag)
	}
	return nil
}

func (p *putter) Close() error {
	if p.closed {
		return syscall.EINVAL
	}
	if p.buf != nil {

		buf := *p.buf
		if buf.Len() > 0 {
			p.flush()
		}
	}
	p.wg.Wait()
	close(p.ch)
	p.closed = true
	if p.err != nil {
		err := p.abort()
		if err != nil {
			return err
		}
		return p.err
	}
	log.Println("Makes: ", p.makes, "Max queue length: ", q_len_max)

	body, err := xml.Marshal(p.xml)
	if err != nil {
		return err
	}
	b := bytes.NewBuffer(body)
	v := url.Values{}
	v.Set("uploadId", p.UploadId)
	req, err := http.NewRequest("POST", p.url.String()+"?"+v.Encode(), b)
	if err != nil {
		return err
	}
	p.b.Sign(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newRespError(resp)
	}
	defer resp.Body.Close()
	// Check md5 hash of concatenated part md5 hashes against ETag
	// more info: https://forums.aws.amazon.com/thread.jspa?messageID=456442&#456442
	calculatedMd5ofParts := fmt.Sprintf("%x", p.md5OfParts.Sum(nil))
	// Parse etag from body of response
	err = xml.NewDecoder(resp.Body).Decode(p)
	if err != nil {
		return err
	}
	// strip part count from end and '"' from front.
	remoteMd5ofParts := strings.Split(p.ETag, "-")[0]
	remoteMd5ofParts = remoteMd5ofParts[1:len(remoteMd5ofParts)]
	if calculatedMd5ofParts != remoteMd5ofParts {
		// TODO: Delete file from S3?
		if err != nil {
			return err
		}
		return fmt.Errorf("MD5 hash of part hashes comparison failed. Hash from multipart complete header: %s."+
			" Calculated multipart hash: %s.", remoteMd5ofParts, calculatedMd5ofParts)
	}
	//log.Println("Hash from multipart complete header:", remoteMd5ofParts)
	//log.Println("Calculated multipart hash:", calculatedMd5ofParts)
	return nil
}

func (p *putter) abort() (err error) {
	v := url.Values{}
	v.Set("uploadId", p.UploadId)
	s := p.url.String() + "?" + v.Encode()
	req, err := http.NewRequest("DELETE", s, nil)
	if err != nil {
		return err
	}
	p.b.Sign(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return newRespError(resp)
	}
	return nil
}
func (p *putter) get_buffer() *bytes.Buffer {
	var b *bytes.Buffer
	if p.makes < p.concurrency*2 && len(p.get) == 0 {
		size := p.bufsz + 1*kb
		s := make([]byte, 0, size)
		b = bytes.NewBuffer(s)
		p.makes++
	} else {
		b = <-p.get
	}
	return b
}

// old buffer pooling code
/////////////////////////////////////
type queued struct {
	when   time.Time
	buffer *bytes.Buffer
}

func makeBuffer(size int64) []byte {
	return make([]byte, 0, size)
}

//debug
var q_len_max int

func (p *putter) makeRecycler() (get, give chan *bytes.Buffer) {
	get = make(chan *bytes.Buffer)
	give = make(chan *bytes.Buffer)

	go func() {
		q := new(list.List)
		for {
			if q.Len() == 0 {
				size := p.bufsz + 1*kb
				q.PushFront(queued{when: time.Now(), buffer: bytes.NewBuffer(makeBuffer(int64(size)))})
				//q.PushFront(queued{when: time.Now(), buffer: bytes.NewBuffer(nil)})
				//log.Println("Make buffer:", size)
				p.makes++
			}

			e := q.Front()

			timeout := time.NewTimer(time.Minute)
			select {
			case b := <-give:
				timeout.Stop()
				q.PushFront(queued{when: time.Now(), buffer: b})
				q_len_max = max(q_len_max, q.Len())

			case get <- e.Value.(queued).buffer:
				timeout.Stop()
				q.Remove(e)
			// free unused buffers
			case <-timeout.C:
				e := q.Front()
				for e != nil {
					n := e.Next()
					if time.Since(e.Value.(queued).when) > time.Minute {
						q.Remove(e)
						e.Value = nil
					}
					e = n
				}
			}
		}

	}()
	return
}
