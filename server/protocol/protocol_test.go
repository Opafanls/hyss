package protocol

import "testing"

func TestBytesCut(t *testing.T) {
	bs := make([]byte, 10)
	for i := 0; i < len(bs); i++ {
		bs[i] = byte(i)
	}
	bs0 := bs[0:1]
	bs1 := bs[1:2]
	t.Logf("%+v   %+v", bs0, bs1)
}
