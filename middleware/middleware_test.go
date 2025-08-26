package middleware

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

var i int

func TestChain(t *testing.T) {

	next := func(_ context.Context, req any) (any, error) { // Handler

		if req != "hello kratos!" {
			t.Errorf("expect %v, got %v", "hello kratos!", req)
		}

		i += 10

		return "reply", nil

	}

	// HANDLER嵌套 -- 对NEXT这个HANDER进行包装嵌套之后（还是HANDLER）调用执行
	got, err := Chain(test1Middleware, test2Middleware, test3Middleware)(next)(context.Background(), "hello kratos!")

	if err != nil {
		t.Errorf("expect %v, got %v", nil, err)
	}

	if !reflect.DeepEqual(got, "reply") {
		t.Errorf("expect %v, got %v", "reply", got)
	}

	if !reflect.DeepEqual(i, 16) {
		t.Errorf("expect %v, got %v", 16, i)
	}

}

// *********************************************************************************************************************
func test1Middleware(handler Handler) Handler {

	return func(ctx context.Context, req any) (reply any, err error) { // Handler

		fmt.Println("test1 before")

		i++

		reply, err = handler(ctx, req) // 对应下一个中间件中返回的HNADLER

		fmt.Println("test1 after")

		return

	}

}

func test2Middleware(handler Handler) Handler {

	return func(ctx context.Context, req any) (reply any, err error) { // Handler

		fmt.Println("test2 before")

		i += 2

		reply, err = handler(ctx, req) // 对应下一个中间件中返回的HNADLER

		fmt.Println("test2 after")

		return

	}

}

func test3Middleware(handler Handler) Handler {

	return func(ctx context.Context, req any) (reply any, err error) { // Handler

		fmt.Println("test3 before")

		i += 3

		reply, err = handler(ctx, req) // 对应目标HANDLER

		fmt.Println("test3 after")

		return

	}

}

// *********************************************************************************************************************
