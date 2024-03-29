package log

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Logrus struct {
	l *logrus.Logger
}

func (l *Logrus) Init(param interface{}) error {
	l.l = logrus.New()
	return nil
}

func (l *Logrus) Tracef(ctx context.Context, format string, args ...interface{}) {
	l.l.Tracef(getLogIDFormat(ctx, format), args...)
}

func (l *Logrus) Debugf(ctx context.Context, format string, args ...interface{}) {
	l.l.Debugf(getLogIDFormat(ctx, format), args...)
}

func (l *Logrus) Infof(ctx context.Context, format string, args ...interface{}) {
	l.l.Infof(getLogIDFormat(ctx, format), args...)
}

func (l *Logrus) Warnf(ctx context.Context, format string, args ...interface{}) {
	l.l.Warnf(getLogIDFormat(ctx, format), args...)
}

func (l *Logrus) Errorf(ctx context.Context, format string, args ...interface{}) {
	l.l.Errorf(getLogIDFormat(ctx, format), args...)
}
