package middleware

import (
	"context"
)

// Handler defines the handler invoked by Middleware.
type Handler func(ctx context.Context, req any) (any, error)

// Middleware is HTTP/gRPC transport middleware.
type Middleware func(Handler) Handler

// Chain returns a Middleware that specifies the chained handler for endpoint.
func Chain(m ...Middleware) Middleware { // 将多个中间件组合成一个中间件（开始的中间件会最后执行）

	return func(next Handler) Handler {

		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}

		return next

	}

}
