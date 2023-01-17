package bitio

import "testing"

func TestByteMake(t *testing.T) {
	buf := make([]byte, 0, 1)

	t.Logf("%+v", buf)
}
