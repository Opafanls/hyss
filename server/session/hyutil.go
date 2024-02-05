package session

import (
	"fmt"
	"time"
)

func SinkID(sourceID string) string {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return fmt.Sprintf("%s:%s:%s", sourceID, "sink", id)
}
