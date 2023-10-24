package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

//Gob是Go自己的以二进制形式序列化和反序列化程序数据的格式，这种数据格式简称之为Gob (Go binary)。

type GobCodec struct {
	conn io.ReadWriteCloser
	buf *bufio.Writer
	dec *gob.Decoder
	enc *gob.Encoder

}

/* 这是Go语言中的一个常用技巧，用于在编译时检查某个类型是否实现了某个接口。这行代码不会影响程序的运行时行为，但它确保了在编译时进行检查。
不会真正创建一个变量。重要的是，如果GobCodec没有实现Codec接口，这行代码会在编译时引发一个错误。 */
var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf: buf,
		dec: gob.NewDecoder(conn),
		enc: gob.NewEncoder(buf),
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}