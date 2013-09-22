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
	url      url.URL
	client   *http.Client
	conf     *s3util.Config
	bufsz    int64
	err      error
	wg       sync.WaitGroup
	get_buf  chan *bytes.Buffer
	give_buf chan *bytes.Buffer

	cur_chunk      int
	content_length int64
	chunk_total    int
	chunks         []*chunk
	get_ch         chan *chunk
	read_ch        chan *chunk

	concurrency int64
	nTry        int
}

type chunk struct {
	id             int
	header         http.Header
	start          int64
	content_length int64
	b              *bytes.Buffer
	len            int64
}

type chunkSlice []chunk

// Methods required to sort
func (c chunkSlice) Len() int {
	return len(c)
}

func (c chunkSlice) Less(i, j chunk) bool {
	return i.id < j.id
}

func (c chunkSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (g *getter) newGetter(url url.URL, c *s3util.Config) (io.ReadCloser, error) {

	// initialize getter
	g = new(getter)
	g.conf = c
	g.url = url
	g.bufsz = buffer_size
	g.get_buf, g.give_buf = makeRecycler()
	g.get_ch = make(chan *chunk)
	g.read_ch = make(chan *chunk)
	// get content length
	r := http.Request{
		Method: "HEAD",
		URL:    &url,
		Body:   nil,
	}
	g.conf.Sign(&r, *g.conf.Keys)
	g.client = g.conf.Client

	resp, err := g.client.Do(&r)
	if err != nil {
		return nil, err
	}
	if resp.Status != "200 OK" {
		return nil, fmt.Errorf("Expected HTTP Status 200, received %q", resp.Status)
	}
	g.content_length = resp.ContentLength
	go func() {
		// init chunks
		for i := int64(0); i < g.content_length; {
			content_length := min(g.bufsz, g.content_length-i)
			c := &chunk{g.chunk_total, nil, content_length, g.bufsz, nil, 0}
			g.chunks = append(g.chunks, c)
			i += g.bufsz
			g.chunk_total++
			g.wg.Add(1)

			// put on get chan
			g.get_ch <- c

		}
	}()

	g.concurrency = min(int64(g.conf.Concurrency), (g.content_length / buffer_size))

	for i := int64(0); i < g.concurrency; i++ {
		go g.worker()
	}

	return g, nil
}

func (g *getter) worker() {
	for c := range g.get_ch {
		g.retryGetChunk(c)
	}

}

func (g *getter) retryGetChunk(c chunk) {

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

func (g *getter) Read(p []byte) (int, error) {
	return 0, nil
}

func (g *getter) Close() error {
	return nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
