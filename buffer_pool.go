package s3gof3r

import (
	"bytes"
	//"log"
)

//debug
var Makes int

var get_buf = make(chan *bytes.Buffer)
var give_buf = make(chan *bytes.Buffer, 10)

func bufferPool() {
	for {
		b := <-give_buf

		select {
		// return to get_ch if room
		case get_buf <- b:

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
