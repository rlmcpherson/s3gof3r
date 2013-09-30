package s3gof3r

import (
	"bytes"
	"log"
)

//debug
var Makes int

var get_buf = make(chan *bytes.Buffer, 10)
var give_buf = make(chan *bytes.Buffer)

func bufferPool() {
	//var b *bytes.Buffer
	for {
		b := <-give_buf

		log.Println("Buffer returned")
		select {
		case get_buf <- b:
			log.Println("Buffers in list", len(give_buf))
		default:
			// do nothing, buffer is garbage collected}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
