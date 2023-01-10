package protocol

import (
	"context"
)

type Handler interface {
	OnInit(ctx context.Context)
	OnMedia(ctx context.Context, mediaType MediaDataType, data interface{}) error
	OnClose() error
}
