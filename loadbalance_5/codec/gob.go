package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	bufr *bufio.Reader
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	bufr := bufio.NewReader(conn)
	buf := bufio.NewWriter(conn)
	return &GobCodec {
		conn: conn,
		bufr: bufr,
		buf: buf,
		dec: gob.NewDecoder(bufr),
		enc: gob.NewEncoder(buf),
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}



func (c *GobCodec) Write(h *Header, body interface {}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error: gob error encoding header:", err)
	}
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
	}
	return 
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}