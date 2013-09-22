package s3gof3r

import (
	"bytes"
	"container/list"
	"log"
	"time"
)

//debug
var makes int
var q_len_max int

type queued struct {
	when   time.Time
	buffer *bytes.Buffer
}

func makeRecycler() (get, give chan *bytes.Buffer) {
	get = make(chan *bytes.Buffer)
	give = make(chan *bytes.Buffer)

	go func() {
		q := new(list.List)
		for {
			if q.Len() == 0 {

				//q.PushFront(queued{when: time.Now(), buffer: bytes.NewBuffer(makeBuffer(int64(u.bufsz)))})
				q.PushFront(queued{when: time.Now(), buffer: bytes.NewBuffer(nil)})
				//log.Println("Make buffer:", u.bufsz)
				makes++
			}

			e := q.Front()

			timeout := time.NewTimer(time.Minute)
			select {
			case b := <-give:
				timeout.Stop()
				q.PushFront(queued{when: time.Now(), buffer: b})
				//log.Println("Return buffer")
				q_len_max = max(q_len_max, q.Len())

			case get <- e.Value.(queued).buffer:
				timeout.Stop()
				q.Remove(e)
				//log.Println("Get buffer")

			case <-timeout.C:
				e := q.Front()
				for e != nil {
					n := e.Next()
					if time.Since(e.Value.(queued).when) > time.Minute {
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
