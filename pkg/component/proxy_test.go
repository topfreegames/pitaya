package component

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

type testHandler struct {
}

type testReq struct {
}

type testResp struct {
}

type testNotify struct {
}

func (t *testHandler) TestFunc(ctx context.Context, req *testReq) (*testResp, error) {
	fmt.Println("call TestFunc")
	return nil, nil
}

func (t *testHandler) TestNotify(ctx context.Context, notice *testNotify) {
	fmt.Println("call TestNotify")
}

func TestNewMethodProxyContext(t *testing.T) {
	h := &testHandler{}
	typ := reflect.TypeOf(h)
	for i := range typ.NumMethod() {
		f := typ.Method(i)

		ctx := NewMethodProxyContext(f, nil, nil)
		ctx.Next()
	}
}
