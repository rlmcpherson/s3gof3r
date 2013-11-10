// TODO: Add documentation
package s3gof3r

import (
	"bytes"
	"fmt"
	//"github.com/rlmcpherson/s3/s3util"
	"io"
	"log"
	"net/http"
	"net/url"
	//"os"
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
	buffer_size     = 20 * MB
	makes_over_conc = 5
)

type getter struct {
	url    url.URL
	client *http.Client
	conf   *Config
	b      *Bucket
	bufsz  int64
	err    error
	wg     sync.WaitGroup

	cur_chunk_id   int
	cur_chunk      *chunk
	content_length int64
	chunk_total    int
	read_ch        chan *chunk
	get_ch         chan *chunk

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

//type bufferpool struct {
//get  chan *bytes.Buffer
//give chan *bytes.Buffer
//}

//func s3Get(raw_url string, c *Config) (io.ReadCloser, http.Header, error) {

//p_url, err := url.Parse(raw_url)
//if err != nil {
//return nil, nil, err
//}

//return newGetter(*p_url, c)
//}

func newGetter(p_url url.URL, c *Config, b *Bucket) (io.ReadCloser, http.Header, error) {

	// initialize getter
	g := new(getter)
	g.conf = c
	g.url = p_url
	g.bufsz = buffer_size
	//g.bp.get, g.bp.give = makeRecycler()
	g.get_ch = make(chan *chunk)
	g.read_ch = make(chan *chunk)
	g.nTry = 5
	g.q_wait = make(map[int]*chunk)

	//start buffer pool
	go bufferPool()

	// get content length
	r, err := http.NewRequest("HEAD", p_url.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	r.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	r.Header.Set("User-Agent", "s3Gof3r")

	g.b.Sign(r)
	g.client = g.conf.Client

	resp, err := g.client.Do(r)
	if err != nil {
		return nil, nil, err
	}
	if resp.Status != "200 OK" {
		return nil, nil, fmt.Errorf("Expected HTTP Status 200, received %q", resp.Status)
	}
	g.content_length = resp.ContentLength

	g.concurrency = min(int64(g.conf.Concurrency), (g.content_length/buffer_size)+1)
	log.Println("Concurrency:", g.concurrency)

	for i := int64(0); i < g.concurrency; i++ {
		go g.worker()
	}
	go g.init_chunks()

	log.Println("End of initialize")
	return g, resp.Header, nil
}

func (g *getter) init_chunks() {
	for i := int64(0); i < g.content_length; {
		size := min(g.bufsz, g.content_length-i)
		c := &chunk{
			id: g.chunk_total,
			header: http.Header{
				"Range": {fmt.Sprintf("bytes=%d-%d",
					i, i+size-1)},
				"User-Agent": {"S3Gof3r"},
				"Date":       {time.Now().UTC().Format(http.TimeFormat)}},

			start: i,
			size:  size,
			b:     nil,
			len:   0}

		i += size
		g.chunk_total++
		g.wg.Add(1)

		// put on get chan
		log.Println("Sending chunk ", c.id, " Offset: ", c.start, " Size:", c.size)
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
	// get buffer to write
	c.b = g.get_buffer()
	for i := 0; i < g.nTry; i++ {
		err = g.getChunk(c)
		if err == nil {
			return
		}
		log.Println(err)
	}
	g.err = err

}

func (g *getter) get_buffer() *bytes.Buffer {
	var b *bytes.Buffer
	// not threadsafe, but worst case will wait for buffer
	if Makes < g.concurrency+makes_over_conc && len(get_buf) < 1 {
		size := g.bufsz + 1*KB
		empty := make([]byte, 0, size)

		b = bytes.NewBuffer(empty)
		//b.Grow(int(g.bufsz))
		Makes++
	} else {
		select {
		case b = <-get_buf:
		}
	}
	return b
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

	resp, err := g.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.Status != "206 Partial Content" {
		//resp.Write(os.Stderr)
		return fmt.Errorf("Expected HTTP Status 206, received %q",
			resp.Status)
	}
	//resp.Header.Write(os.Stderr)
	log.Println("buffer size: ", c.b.Len())

	//n, err := io.Copy(c.b, resp.Body) //TODO: Md5 checking
	n, err := c.b.ReadFrom(resp.Body)
	//n, err := io.CopyN(c.b, resp.Body, c.size-1)
	log.Println("Body length:", c.b.Len())
	//c.b.Write(b)
	if err != nil {
		return err
	}
	if n != c.size {
		log.Println("Size mismatch")
		return fmt.Errorf("Chunk %d: Expected %d bytes, received %d",
			c.id, c.size, n)
	}

	g.read_ch <- c
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
	if g.cur_chunk == nil {
		g.cur_chunk, err = g.get_cur_chunk()
		if err != nil {
			return 0, err
		}
	}
	n, err := g.cur_chunk.b.Read(p)

	// Empty buffer, move on to next
	if err == io.EOF {
		log.Printf("Completed chunk %d of %d\n", g.cur_chunk.id, g.chunk_total-1)
		// Do not send EOF for each chunk.
		if g.cur_chunk.id == g.chunk_total-1 && g.cur_chunk.b.Len() == 0 {
			log.Println("Last Chunk:", g.cur_chunk)
			return 0, io.EOF
		}
		give_buf <- g.cur_chunk.b
		g.cur_chunk = nil
		g.cur_chunk_id++
		return n - 1, nil
	}

	return n, err
}

func (g *getter) get_cur_chunk() (*chunk, error) {
	var cur_chunk *chunk
	var err error
	for {
		// first check q_wait
		if cur_chunk, ok := g.q_wait[g.cur_chunk_id]; ok {
			delete(g.q_wait, g.cur_chunk_id)
			//log.Println("return cur_chunk:", cur_chunk)
			return cur_chunk, err
		}
		// if not present, read from channel
		cur_chunk = <-g.read_ch
		g.q_wait[cur_chunk.id] = cur_chunk

	}
	return cur_chunk, err
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
	log.Println("makes:", Makes)
	//log.Println("max q len:", Q_max)

	return nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
