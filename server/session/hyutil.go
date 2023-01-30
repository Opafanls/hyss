package session

import (
	"fmt"
	"github.com/Opafanls/hylan/server/constdef"
	"time"
)

func BaseSessionType(s HySessionI) constdef.SessionType {
	sessType, exist := s.GetConfig(constdef.ConfigKeySessionType)
	if exist {
		return constdef.SessionType(sessType.(int))
	}
	return constdef.SessionNotFoundAtBase
}

func SinkID(sourceID string) string {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return fmt.Sprintf("%s:%s:%s", sourceID, "sink", id)
}
