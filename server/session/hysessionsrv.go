package session

func Serve(sess HySessionI, h ProtocolHandler) {
	sess.SetHandler(h)
	_ = sess.Cycle()
}
