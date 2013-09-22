//-------------Downloader
package s3gof3r

import (
	"bytes"
	"fmt"
	"github.com/rlmcpherson/s3/s3util"
	"io"
	"net/http"
	"net/url"
	"sync"
	"syscall"
	"time"
)

const (
	_        = iota
	KB int64 = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
)

const (
	buffer_size = 20 * MB
)

type getter struct {
	url    url.URL
	client *http.Client
	conf   *s3util.Config
	bufsz  int64
	err    error
	wg     sync.WaitGroup

	cur_chunk_id   int
	cur_chunk      *chunk
	content_length int64
	chunk_total    int
	get_ch         chan *chunk
	read_ch        chan *chunk

	bp bufferpool

	q_wait map[int]*chunk

	concurrency int64
	nTry        int
	closed      bool
}

type chunk struct {
	id     int
	header http.Header
	start  int64
	size   int64
	b      *bytes.Buffer
	len    int64
}

type bufferpool struct {
	get  chan *bytes.Buffer
	give chan *bytes.Buffer
}

//type ChunkSlice []*chunk

//Methods required to sort
//func (c ChunkSlice) Len() int {
//return len(c)
//}

//func (c ChunkSlice) Less(i, j *chunk) bool {
//return i.id < j.id
//}

//func (c ChunkSlice) Swap(i, j *chunk) {
//c[i], c[j] = c[j], c[i]
//}

func Open(raw_url string, c *s3util.Config) (io.ReadCloser, http.Header, error) {

	p_url, err := url.Parse(raw_url)
	if err != nil {
		return nil, nil, err
	}

	return newGetter(*p_url, c)
}

func newGetter(url url.URL, c *s3util.Config) (io.ReadCloser, http.Header, error) {

	// initialize getter
	g := new(getter)
	g.conf = c
	g.url = url
	g.bufsz = buffer_size
	g.bp.get, g.bp.give = makeRecycler()
	g.get_ch = make(chan *chunk)
	g.read_ch = make(chan *chunk)

	// get content length
	r := http.Request{
		Method: "HEAD",
		URL:    &g.url,
		Body:   nil,
	}
	r.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	r.Header.Set("User-Agent", "s3Gof3r")

	g.conf.Sign(&r, *g.conf.Keys)
	g.client = g.conf.Client

	resp, err := g.client.Do(&r)
	if err != nil {
		return nil, nil, err
	}
	if resp.Status != "200 OK" {
		return nil, nil, fmt.Errorf("Expected HTTP Status 200, received %q", resp.Status)
	}
	g.content_length = resp.ContentLength

	g.concurrency = min(int64(g.conf.Concurrency), (g.content_length / buffer_size))

	for i := int64(0); i < g.concurrency; i++ {
		go g.worker()
	}

	return g, resp.Header, nil
}

func (g *getter) init_chunks() {
	for i := int64(0); i < g.content_length; {
		size := min(g.bufsz, g.content_length-i)
		c := &chunk{
			id: g.chunk_total,
			header: http.Header{
				"Range": {fmt.Sprintf("bytes=%d-%d",
					i, size)}}, //TODO: add time, agent
			start: i,
			size:  size,
			b:     nil,
			len:   0}

		//g.chunks = append(g.chunks, c)
		i += g.bufsz
		g.chunk_total++
		g.wg.Add(1)

		// put on get chan
		g.get_ch <- c
	}

}

func (g *getter) worker() {
	for c := range g.get_ch {
		g.retryGetChunk(c)
	}

}

func (g *getter) retryGetChunk(c *chunk) {

	defer g.wg.Done()
	var err error
	for i := 0; i < g.nTry; i++ {
		err = g.getChunk(c)
		if err == nil {
			return
		}
	}
	g.err = err

}

func (g *getter) getChunk(c *chunk) error {

	// get buffer to write
	c.b = <-g.bp.get
	c.b.Reset()

	r := http.Request{
		Method: "GET",
		URL:    &g.url,
		Body:   nil,
		Header: c.header,
	}
	g.conf.Sign(&r, *g.conf.Keys)

	resp, err := g.client.Do(&r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.Status != "206 Partial Content" {
		return fmt.Errorf("Expected HTTP Status 206, received %q",
			resp.Status)
	}
	n, err := io.Copy(c.b, resp.Body) //TODO: Md5 checking
	if err != nil {
		return err
	}
	if n != c.size {
		return fmt.Errorf("Chunk %d: Expected %d bytes, received %d",
			c.id, c.size, n)
	}
	g.read_ch <- c
	return nil
}

func (g *getter) Read(p []byte) (int, error) {
	if g.closed {
		return 0, syscall.EINVAL
	}
	if g.err != nil {
		return 0, g.err
	}
	if g.cur_chunk == nil {
		if err := g.get_cur_chunk; err != nil {
			return 0, g.err
		}
	}
	n, err := g.cur_chunk.b.Read(p)

	// Empty buffer, move on to next
	if err == io.EOF {
		g.bp.give <- g.cur_chunk.b
		g.cur_chunk = nil
		g.cur_chunk_id++
	}

	return n, err
}

func (g *getter) get_cur_chunk() (err error) {
	var cur_chunk *chunk

	for g.cur_chunk == nil {
		// first check q_wait
		if cur_chunk, ok := g.q_wait[g.cur_chunk_id]; ok {
			g.cur_chunk = cur_chunk
			delete(g.q_wait, g.cur_chunk_id)
		}
		// if not present, read from channel
		cur_chunk = <-g.read_ch
		g.q_wait[cur_chunk.id] = cur_chunk

	}
	return err
}

func (g *getter) Close() error {
	if g.closed {
		return syscall.EINVAL
	}
	if g.err != nil {
		return g.err
	}
	g.wg.Wait()
	close(g.read_ch)
	close(g.get_ch)
	close(g.bp.give)
	close(g.bp.get)

	g.closed = true

	return nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
