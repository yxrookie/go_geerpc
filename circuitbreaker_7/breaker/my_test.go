package breaker

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/breaker"
)

type mockError struct {
    status int
}

func (e mockError) Error() string {
    return fmt.Sprintf("HTTP STATUS: %d", e.status)
}

func TestMy(t *testing.T) {
    for i := 0; i < 1000; i++ {
        if err := breaker.DoWithFallback("test", func() error {
            return mockRequest()
        }, func(err error) error {
            // 发生了熔断，这里可以自定义熔断错误转换
            return errors.New("当前服务不可用，请稍后再试")
        }); err != nil {
            println(err.Error())
        }
    }
}

func mockRequest() error {
    source := rand.NewSource(time.Now().UnixNano())
    r := rand.New(source)
    num := r.Intn(100)
    if num%4 == 0 {
        return nil
    } else if num%5 == 0 {
        return mockError{status: 500}
    }
    return errors.New("dummy")
}
