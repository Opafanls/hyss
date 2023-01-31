package cache

import (
	"testing"
	"time"
)

func TestRing0(t *testing.T) {
	ring := NewRing0(1)
	ring.Reset()
	go func() {
		data, exist := ring.Pull(false)

		if !exist {
			t.Fatal("should exist")
		}

		t.Logf("%s", data)

		data, exist = ring.Pull(false)

		if exist {
			t.Fatal("should not exist")
		}

		t.Logf("%s", data)
	}()
	ring.Push("1")
	ring.Push("2")
	time.Sleep(time.Second)
}

func TestRing0Idx(t *testing.T) {
	r := NewRing0(16)
	for i := 0; i < 20; i++ {
		idx := r.Push(i)
		t.Logf("%d", idx)
	}
	t.Logf("---")
	for i := 0; i < 20; i++ {
		data, exist := r.Pull(false)
		if exist {
			t.Logf("%v", data)
		}
	}
}

func TestRing0IdxSetReadIndex(t *testing.T) {
	r := NewRing0(16)
	for i := 0; i < 16; i++ {
		idx := r.Push(i)
		t.Logf("%d", idx)
	}
	t.Logf("---")
	r.SetReadIdx(8)
	for i := 0; i < 16; i++ {
		data, exist := r.Pull(false)
		if exist {
			t.Logf("%v", data)
		}
	}
}
