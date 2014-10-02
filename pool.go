package s3gof3r

import (
	"bytes"
	"container/list"
	"time"
)

type qBuf struct {
	when   time.Time
	buffer *bytes.Buffer
}

type bp struct {
	makes    int
	get      chan *bytes.Buffer
	give     chan *bytes.Buffer
	quit     chan struct{}
	timeout  time.Duration
	makeSize int64
}

func makeBuffer(size int64) []byte {
	return make([]byte, 0, size)
}

func newBufferPool(bufsz int64) (np *bp) {
	np = &bp{
		get:      make(chan *bytes.Buffer),
		give:     make(chan *bytes.Buffer),
		quit:     make(chan struct{}),
		timeout:  time.Minute,
		makeSize: bufsz,
	}
	go func() {
		q := new(list.List)
		for {
			if q.Len() == 0 {
				size := np.makeSize + 100*kb // allocate overhead to avoid slice growth
				q.PushFront(qBuf{when: time.Now(), buffer: bytes.NewBuffer(makeBuffer(size))})
				np.makes++
				logger.debugPrintf("buffer %d of %d MB allocated", np.makes, np.makeSize/(1*mb))
			}

			e := q.Front()

			timeout := time.NewTimer(np.timeout)
			select {
			case b := <-np.give:
				timeout.Stop()
				b.Reset()
				q.PushFront(qBuf{when: time.Now(), buffer: b})

			case np.get <- e.Value.(qBuf).buffer:
				timeout.Stop()
				q.Remove(e)

			case <-timeout.C:
				// free unused buffers
				e := q.Front()
				for e != nil {
					n := e.Next()
					if time.Since(e.Value.(qBuf).when) > np.timeout {
						q.Remove(e)
						e.Value = nil
					}
					e = n
				}
			case <-np.quit:
				logger.debugPrintf("%d buffers of %d MB allocated", np.makes, np.makeSize/(1*mb))
				return
			}
		}

	}()
	return np
}
