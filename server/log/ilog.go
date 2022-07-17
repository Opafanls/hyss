package log

type ILog interface {
	Init(param interface{}) error
	Info()
	Error()
}
