package s3gof3r

import (
	"encoding/xml"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func newObjectLister(c *Config, b *Bucket, prefixes []string, maxKeys int) (*ObjectLister, error) {
	l := new(ObjectLister)
	l.c, l.b = new(Config), new(Bucket)
	*l.c, *l.b = *c, *b
	l.c.NTry = max(c.NTry, 1)
	l.c.Concurrency = max(c.Concurrency, 1)
	l.getCh, l.putCh = make(chan string), make(chan []string, 1)
	l.quit = make(chan struct{})
	l.prefixes = prefixes
	l.maxKeys = maxKeys

	for i := 0; i < l.c.Concurrency; i++ {
		l.wg.Add(1)
		go l.worker()
	}
	go l.initPrefixes()

	return l, nil
}

type ObjectLister struct {
	b        *Bucket
	c        *Config
	prefixes []string
	maxKeys  int

	next     []string
	err      error
	getCh    chan string
	putCh    chan []string
	wg       sync.WaitGroup
	quit     chan struct{}
	quitOnce sync.Once
}

func (l *ObjectLister) closeQuit() {
	l.quitOnce.Do(func() { close(l.quit) })
}

func (l *ObjectLister) initPrefixes() {
	// We first enqueue all of the prefixes we were given
	for _, p := range l.prefixes {
		l.getCh <- p
	}
	close(l.getCh)

	l.wg.Wait()
	close(l.putCh)
}

func (l *ObjectLister) worker() {
	for p := range l.getCh {
		var continuation string
	retries:
		for {
			res, err := l.retryListObjects(p, continuation)
			if err != nil {
				select {
				case <-l.quit:
					return
				default:
					l.err = err
					l.closeQuit()
					return
				}
			}

			keys := make([]string, 0, len(res.Contents))
			for _, c := range res.Contents {
				keys = append(keys, c.Key)
			}

			select {
			case <-l.quit:
				return
			case l.putCh <- keys:
				continuation = res.NextContinuationToken
				if continuation != "" {
					continue
				}

				// Break from this prefix and grab the next one
				break retries
			}
		}
	}

	l.wg.Done()
}

func (l *ObjectLister) retryListObjects(p, continuation string) (*listBucketResult, error) {
	var err error
	var res *listBucketResult
	for i := 0; i < l.c.NTry; i++ {
		opts := listObjectsOptions{MaxKeys: l.maxKeys, Prefix: p, ContinuationToken: continuation}
		res, err = listObjects(l.c, l.b, opts)
		if err == nil {
			return res, nil
		}

		time.Sleep(time.Duration(math.Exp2(float64(i))) * 100 * time.Millisecond) // exponential back-off
	}

	return nil, err
}

// Next moves the iterator to the next set of results. It returns true if there
// are more results, or false if there are no more results or there was an
// error.
func (l *ObjectLister) Next() bool {
	if l.err != nil {
		return false
	}

	select {
	case n, ok := <-l.putCh:
		if !ok {
			l.err = nil
			return false
		}

		l.next = n
		return true
	case <-l.quit:
		return false
	}
}

func (l *ObjectLister) Value() []string {
	return l.next
}

func (l *ObjectLister) Error() error {
	return l.err
}

func (l *ObjectLister) Close() {
	l.closeQuit()
}

// ListObjectsOptions specifies the options for a ListObjects operation on a S3
// bucket
type listObjectsOptions struct {
	// Maximum number of keys to return per request
	MaxKeys int
	// Only list those keys that start with the given prefix
	Prefix string
	// Continuation token from the previous request
	ContinuationToken string
}

type listBucketResult struct {
	Name                  string               `xml:"Name"`
	Prefix                string               `xml:"Prefix"`
	KeyCount              int                  `xml:"KeyCount"`
	MaxKeys               int                  `xml:"MaxKeys"`
	IsTruncated           bool                 `xml:"IsTrucated"`
	NextContinuationToken string               `xml:"NextContinuationToken"`
	Contents              []listBucketContents `xml:"Contents"`
}

type listBucketContents struct {
	Key            string         `xml:"Key"`
	LastModified   time.Time      `xml:"LastModified"`
	ETag           string         `xml:"ETag"`
	Size           int64          `xml:"Size"`
	StorageClass   string         `xml:"StorageClass"`
	CommonPrefixes []CommonPrefix `xml:"CommonPrefixes"`
}

type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

type ListObjectsResult struct {
	result *listBucketResult
}

func listObjects(c *Config, b *Bucket, opts listObjectsOptions) (result *listBucketResult, err error) {
	result = new(listBucketResult)
	u, err := b.url("", c)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("list-type", "2")
	if opts.MaxKeys > 0 {
		q.Set("max-keys", strconv.Itoa(opts.MaxKeys))
	}
	if opts.Prefix != "" {
		q.Set("prefix", opts.Prefix)
	}
	if opts.ContinuationToken != "" {
		q.Set("continuation-token", opts.ContinuationToken)
	}
	u.RawQuery = q.Encode()

	r := http.Request{
		Method: "GET",
		URL:    u,
	}
	b.Sign(&r)

	resp, err := b.conf().Do(&r)
	if err != nil {
		return nil, err
	}
	defer checkClose(resp.Body, err)
	if resp.StatusCode != 200 {
		return nil, newRespError(resp)
	}

	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}
