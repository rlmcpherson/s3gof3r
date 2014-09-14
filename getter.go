package s3gof3r

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"sync"
	"syscall"
	"time"
)

const (
	qWaitSz = 2
)

type getter struct {
	url   url.URL
	b     *Bucket
	bufsz int64
	err   error
	wg    sync.WaitGroup

	chunkID    int
	rChunk     *chunk
	contentLen int64
	bytesRead  int64
	chunkTotal int

	readCh chan *chunk
	getCh  chan *chunk
	quit   chan struct{}
	qWait  map[int]*chunk

	bp *bp

	closed bool
	c      *Config

	md5 hash.Hash
}

type chunk struct {
	id     int
	header http.Header
	start  int64
	size   int64
	b      *bytes.Buffer
	len    int64
}

func newGetter(getURL url.URL, c *Config, b *Bucket) (io.ReadCloser, http.Header, error) {
	g := new(getter)
	g.url = getURL
	g.c = c
	g.bufsz = max64(c.PartSize, 1)
	g.c.NTry = max(c.NTry, 1)
	g.c.Concurrency = max(c.Concurrency, 1)

	g.getCh = make(chan *chunk)
	g.readCh = make(chan *chunk)
	g.quit = make(chan struct{})
	g.qWait = make(map[int]*chunk)
	g.b = b
	g.md5 = md5.New()

	// use get instead of head for error messaging
	resp, err := g.retryRequest("GET", g.url.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	defer checkClose(resp.Body, &err)
	if resp.StatusCode != 200 {
		return nil, nil, newRespError(resp)
	}
	g.contentLen = resp.ContentLength
	g.chunkTotal = int((g.contentLen + g.bufsz - 1) / g.bufsz) // round up, integer division
	logger.debugPrintf("object size: %3.2g MB", float64(g.contentLen)/float64((1*mb)))

	g.bp = newBufferPool(g.bufsz)

	for i := 0; i < g.c.Concurrency; i++ {
		go g.worker()
	}
	go g.initChunks()
	return g, resp.Header, nil
}

func (g *getter) retryRequest(method, urlStr string, body io.ReadSeeker) (resp *http.Response, err error) {
	for i := 0; i < g.c.NTry; i++ {
		var req *http.Request
		req, err = http.NewRequest(method, urlStr, body)
		if err != nil {
			return
		}
		g.b.Sign(req)
		resp, err = g.c.Client.Do(req)
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

func (g *getter) initChunks() {
	id := 0
	for i := int64(0); i < g.contentLen; {
		for len(g.qWait) >= qWaitSz {
			// Limit growth of qWait
			time.Sleep(100 * time.Millisecond)
		}
		size := min64(g.bufsz, g.contentLen-i)
		c := &chunk{
			id: id,
			header: http.Header{
				"Range": {fmt.Sprintf("bytes=%d-%d",
					i, i+size-1)},
			},
			start: i,
			size:  size,
			b:     nil,
			len:   0}
		i += size
		id++
		g.wg.Add(1)
		g.getCh <- c
	}
	close(g.getCh)
}

func (g *getter) worker() {
	for c := range g.getCh {
		g.retryGetChunk(c)
	}

}

func (g *getter) retryGetChunk(c *chunk) {
	defer g.wg.Done()
	var err error
	c.b = <-g.bp.get
	for i := 0; i < g.c.NTry; i++ {
		time.Sleep(time.Duration(math.Exp2(float64(i))) * 100 * time.Millisecond) // exponential back-off
		err = g.getChunk(c)
		if err == nil {
			return
		}
		logger.debugPrintf("error on attempt %d: retrying chunk: %v, error: %s", i, c, err)
	}
	g.err = err
	close(g.quit) // out of tries, ensure quit by closing channel
}

func (g *getter) getChunk(c *chunk) error {
	// ensure buffer is empty
	c.b.Reset()

	r, err := http.NewRequest("GET", g.url.String(), nil)
	if err != nil {
		return err
	}
	r.Header = c.header
	g.b.Sign(r)
	resp, err := g.c.Client.Do(r)
	if err != nil {
		return err
	}
	defer checkClose(resp.Body, &err)
	if resp.StatusCode != 206 {
		return newRespError(resp)
	}
	n, err := c.b.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	if n != c.size {
		return fmt.Errorf("chunk %d: Expected %d bytes, received %d",
			c.id, c.size, n)
	}
	g.readCh <- c
	return nil
}

func (g *getter) Read(p []byte) (int, error) {
	var err error
	if g.closed {
		return 0, syscall.EINVAL
	}
	if g.err != nil {
		return 0, g.err
	}
	if g.rChunk == nil {
		g.rChunk, err = g.nextChunk()
		if err != nil {
			return 0, err
		}
	}

	n, err := g.rChunk.b.Read(p)
	if g.c.Md5Check {
		g.md5.Write(p[0:n])
	}

	// Empty buffer, move on to next
	if err == io.EOF {
		// Do not send EOF for each chunk.
		if !(g.rChunk.id == g.chunkTotal-1 && g.rChunk.b.Len() == 0) {
			err = nil
		}
		g.bp.give <- g.rChunk.b // recycle buffer
		g.rChunk = nil
		g.chunkID++
	}
	g.bytesRead = g.bytesRead + int64(n)
	return n, err
}

func (g *getter) nextChunk() (*chunk, error) {
	for {

		// first check qWait
		c := g.qWait[g.chunkID]
		if c != nil {
			delete(g.qWait, g.chunkID)
			return c, nil
		}
		// if next chunk not in qWait, read from channel
		select {
		case c := <-g.readCh:
			g.qWait[c.id] = c
		case <-g.quit:
			return nil, g.err // fatal error, quit.
		}
	}
}

func (g *getter) Close() error {
	if g.closed {
		return syscall.EINVAL
	}
	if g.err != nil {
		return g.err
	}
	g.wg.Wait()
	g.closed = true
	close(g.bp.quit)
	if g.bytesRead != g.contentLen {
		return fmt.Errorf("read error: %d bytes read. expected: %d", g.bytesRead, g.contentLen)
	}
	if g.c.Md5Check {
		if err := g.checkMd5(); err != nil {
			return err
		}
	}
	return nil
}

func (g *getter) checkMd5() (err error) {
	calcMd5 := fmt.Sprintf("%x", g.md5.Sum(nil))
	md5Path := fmt.Sprint(".md5", g.url.Path, ".md5")
	md5Url, err := g.b.url(md5Path)
	if err != nil {
		return err
	}

	logger.debugPrintln("md5: ", calcMd5)
	logger.debugPrintln("md5Path: ", md5Path)
	resp, err := g.retryRequest("GET", md5Url.String(), nil)
	if err != nil {
		return
	}
	defer checkClose(resp.Body, &err)
	if resp.StatusCode != 200 {
		return fmt.Errorf("MD5 check failed: %s not found: %s", md5Url.String(), newRespError(resp))
	}
	givenMd5, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if calcMd5 != string(givenMd5) {
		return fmt.Errorf("MD5 mismatch. given:%s calculated:%s", givenMd5, calcMd5)
	}
	return
}
