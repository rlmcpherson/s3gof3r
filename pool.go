package s3gof3r

import (
	"container/list"
	"time"
)

type qb struct {
	when time.Time
	s    []byte
}

type bp struct {
	makes   int
	get     chan []byte
	give    chan []byte
	quit    chan struct{}
	timeout time.Duration
	bufsz   int64
	sizech  chan int64
}

func bufferPool(bufsz int64) (sp *bp) {
	sp = &bp{
		get:     make(chan []byte),
		give:    make(chan []byte),
		quit:    make(chan struct{}),
		timeout: time.Minute,
		sizech:  make(chan int64),
	}
	go func() {
		q := new(list.List)
		for {
			if q.Len() == 0 {
				q.PushFront(qb{when: time.Now(), s: make([]byte, bufsz)})
				sp.makes++
			}

			e := q.Front()

			timeout := time.NewTimer(sp.timeout)
			select {
			case b := <-sp.give:
				timeout.Stop()
				q.PushFront(qb{when: time.Now(), s: b})
			case sp.get <- e.Value.(qb).s:
				timeout.Stop()
				q.Remove(e)
			case <-timeout.C:
				// free unused slices older than timeout
				e := q.Front()
				for e != nil {
					n := e.Next()
					if time.Since(e.Value.(qb).when) > sp.timeout {
						q.Remove(e)
						e.Value = nil
					}
					e = n
				}
			case sz := <-sp.sizech: // update buffer size, free buffers
				bufsz = sz
			case <-sp.quit:
				logger.debugPrintf("%d buffers of %d MB allocated", sp.makes, bufsz/(1*mb))
				return
			}
		}

	}()
	return sp
}
