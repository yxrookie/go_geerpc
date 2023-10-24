// 实现信息编解码

package codec

import "io"

type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number chosen by client
	Error         string
}

type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error 
	Write(*Header, interface{} ) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (	
	GobType Type = "appliaction/gob"
	JsonType Type = "application/json" // not implemented
)

// 全局映射，根据不同的 Type 获取对应的编解码函数
var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}

