package session

import "github.com/Opafanls/hylan/server/log"

func Serve(sess HySessionI, h ProtocolHandler) {
	sess.SetHandler(h)
	var err error
	defer func() {
		if err != nil {
			log.Errorf(sess.Ctx(), "conn done with err: %+v", err)
		} else {
			log.Infof(sess.Ctx(), "conn done with no err")
		}
		_ = sess.Close()
	}()
	err = sess.Cycle()
}
