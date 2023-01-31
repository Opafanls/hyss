package pool

type IPool interface {
	Init(size int)
	Take() interface{}
	Ret(interface{})
	New() interface{}
}

type DefaultPool struct {
	c chan interface{}
}

func (d *DefaultPool) Init(size int) {
	d.c = make(chan interface{}, size)
	for i := 0; i < size; i++ {
		d.c <- d.New()
	}
}

func (d *DefaultPool) Take() interface{} {
	var r interface{}
	select {
	case item, ok := <-d.c:
		if ok {
			r = item
			return r
		}
	}
	return d.New()
}

func (d *DefaultPool) Ret(i interface{}) {
	select {
	case d.c <- i:
		return
	}
}

func (d *DefaultPool) New() interface{} {
	panic("implements me")
}
