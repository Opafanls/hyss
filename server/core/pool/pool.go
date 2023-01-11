package pool

var defaultPool BufPool = &DefaultBufPool{}

type BufPool interface {
	Make(size int) ([]byte, error)
	Reset()
	Return([]byte)
}

type DefaultBufPool struct {
}

func (d *DefaultBufPool) Make(size int) ([]byte, error) {
	return make([]byte, size), nil
}

func (d *DefaultBufPool) Reset() {

}

func (d *DefaultBufPool) Return(bytes []byte) {

}

func P() BufPool {
	return defaultPool
}
