package writer

import (
	"io"

	"github.com/go-kratos/kratos/v2/log"
)

type Writer struct {
	logger log.Logger
}

var _ io.Writer = (*Writer)(nil)

func New(logger log.Logger) *Writer {
	return &Writer{
		logger: logger,
	}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	return len(p), w.logger.Log(log.LevelInfo, "msg", string(p))
}
