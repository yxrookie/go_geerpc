package breaker

// 空断路器的实现，使不需要断路保护的情况下仍然可以使用 Breaker 对象作为占位符
const nopBreakerName = "nopBreaker"

type nopBreaker struct{}

func newNopBreaker() Breaker {
	return nopBreaker{}
}

func (b nopBreaker) Name() string {
	return nopBreakerName
}

func (b nopBreaker) Allow() (Promise, error) {
	return nopPromise{}, nil
}

func (b nopBreaker) Do(req func() error) error {
	return req()
}

func (b nopBreaker) DoWithAcceptable(req func() error, _ Acceptable) error {
	return req()
}

func (b nopBreaker) DoWithFallback(req func() error, _ func(err error) error) error {
	return req()
}

func (b nopBreaker) DoWithFallbackAcceptable(req func() error, _ func(err error) error,
	_ Acceptable) error {
	return req()
}

type nopPromise struct{}

func (p nopPromise) Accept() {
}

func (p nopPromise) Reject(_ string) {
}
