package eventbus

//总线处理
type IBus interface {
	Register(register Register) int
	UniCast()
	Broadcast()
}

//注册器
type Register interface {
}
