package s3gof3r

import (
	"container/list"
	"time"
)

type qs struct {
	when time.Time
	s    []byte
}

type sp struct {
	makes   int
	get     chan []byte
	give    chan []byte
	quit    chan struct{}
	timeout time.Duration
	bufsz   int64
	sizech  chan int64
}

func newSlicePool(bufsz int64) (np *sp) {
	np = &sp{
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
				q.PushFront(qs{when: time.Now(), s: make([]byte, bufsz)})
				np.makes++
				logger.debugPrintf("s %d of %d MB allocated", np.makes, bufsz/(1*mb))
			}

			e := q.Front()

			timeout := time.NewTimer(np.timeout)
			select {
			case b := <-np.give:
				timeout.Stop()
				q.PushFront(qs{when: time.Now(), s: b})
			case np.get <- e.Value.(qs).s:
				timeout.Stop()
				q.Remove(e)
			case <-timeout.C:
				// free unused slices older than timeout
				e := q.Front()
				for e != nil {
					n := e.Next()
					if time.Since(e.Value.(qs).when) > np.timeout {
						q.Remove(e)
						e.Value = nil
					}
					e = n
				}
			case sz := <-np.sizech: // update buffer size, free buffers
				bufsz = sz
			case <-np.quit:
				logger.debugPrintf("%d buffers of %d MB allocated", np.makes, bufsz/(1*mb))
				return
			}
		}

	}()
	return np
}
