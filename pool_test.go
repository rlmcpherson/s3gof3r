package s3gof3r

import (
	"testing"
	"time"
)

func TestBP(t *testing.T) {

	bp := newBufferPool(kb)
	bp.timeout = 1 * time.Millisecond
	b := <-bp.get
	if cap(b.Bytes()) != int(kb+100*kb) {
		t.Errorf("Expected buffer capacity: %d. Actual: %d", kb, b.Len())
	}
	bp.give <- b
	if bp.makes != 2 {
		t.Errorf("Expected makes: %d. Actual: %d", 2, bp.makes)
	}

	b = <-bp.get
	bp.give <- b
	time.Sleep(2 * time.Millisecond)
	if bp.makes != 3 {
		t.Errorf("Expected makes: %d. Actual: %d", 3, bp.makes)
	}
	bp.quit <- true
}
