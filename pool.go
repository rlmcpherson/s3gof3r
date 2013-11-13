package s3gof3r

import (
	"bytes"
)

func startBufferPool(size int) (get, give chan *bytes.Buffer) {
	get = make(chan *bytes.Buffer, size)
	give = make(chan *bytes.Buffer)
	go func() {
		for {
			b := <-give
			select {
			case get <- b:
				// buffer is returned to the pool
			default:
				// do nothing, buffer is garbage collected}
			}
		}
	}()
	return
}
