package s3gof3r

import (
	"bytes"
	"container/list"
	"log"
	"time"
)

//debug
var Makes int
var Q_max int

const (
	bufsz = 5 * 1024 * 1024
)

type queued struct {
	when   time.Time
	buffer *bytes.Buffer
}

func makeBuffer(size int64) []byte {
	return make([]byte, size)
}

func makeRecycler() (get, give chan *bytes.Buffer) {
	get = make(chan *bytes.Buffer)
	give = make(chan *bytes.Buffer)

	go func() {
		q := new(list.List)
		for {
			if q.Len() == 0 {

				//q.PushFront(queued{when: time.Now(), buffer: bytes.NewBuffer(makeBuffer(int64(bufsz)))})
				q.PushFront(queued{when: time.Now(), buffer: bytes.NewBuffer(nil)})
				log.Println("Make buffer")
				Makes++
			}

			e := q.Front()

			timeout := time.NewTimer(time.Second * 10)
			select {
			case b := <-give:
				timeout.Stop()
				q.PushFront(queued{when: time.Now(), buffer: b})
				//log.Println("Return buffer")
				Q_max = max(Q_max, q.Len())

			case get <- e.Value.(queued).buffer:
				timeout.Stop()
				q.Remove(e)
				//log.Println("Get buffer")

			case <-timeout.C:
				e := q.Front()
				for e != nil {
					n := e.Next()
					if time.Since(e.Value.(queued).when) > time.Second*5 {
						q.Remove(e)
						e.Value = nil
						log.Println("Delete old buffer") //TODO: remove logging
					}
					e = n
				}
			}
		}

	}()

	return
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
