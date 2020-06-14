package skiptro

import (
	"context"
	"io"
)

type readerCtx struct {
	ctx context.Context
	r   io.Reader
}

func NewReader(ctx context.Context, r io.Reader) io.Reader {
	if r, ok := r.(*readerCtx); ok && ctx == r.ctx {
		return r
	}
	return &readerCtx{ctx: ctx, r: r}
}

func (r *readerCtx) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}
