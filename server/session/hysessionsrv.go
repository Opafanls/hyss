package session

import "github.com/Opafanls/hylan/server/log"

func Serve(sess HySessionI, h ProtocolHandler) {
	sess.SetHandler(h)
	err := sess.Cycle()
	if err != nil {
		log.Errorf(sess.Ctx(), "Cycle err: %+v", err)
	}
	log.Infof(sess.Ctx(), "session close with err: %+v %+v", sess.Close(), sess.Base())
}
