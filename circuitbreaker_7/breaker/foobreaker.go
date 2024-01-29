package breaker

import (

	"errors"
	"fmt"
)


type FooError struct {
  Breaker
}

func (f FooError) Error() string {
	return "error!!!"
}


func (f FooError) FooDoWithFallBack(fooerr error) {
	if err:= DoWithFallback("test", func() error {
		return fooerr
	}, func(err error) error {
		return errors.New("服务不可用，远程调用失败")
	}); err != nil {
		fmt.Println(err.Error())
	}
} 

