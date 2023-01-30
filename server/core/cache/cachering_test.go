package cache

import (
	"testing"
	"time"
)

func TestRing0(t *testing.T) {
	ring := NewRing0(1)
	ring.Reset()
	go func() {
		data, exist := ring.Pull()

		if !exist {
			t.Fatal("should exist")
		}

		t.Logf("%s", data)

		data, exist = ring.Pull()

		if exist {
			t.Fatal("should not exist")
		}

		t.Logf("%s", data)
	}()
	ring.Push("1")
	ring.Push("2")
	time.Sleep(time.Second)
}
