package frame

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"time"
)

var (
	header = [2]byte{0xAA, 0x55}
	footer = [2]byte{0x55, 0xAA}
)

var (
	ErrBadHeader = errors.New("frame: geçersiz başlık")
	ErrBadFooter = errors.New("frame: geçersiz bitiş")
)

type Conn struct {
	conn    net.Conn
	reader  *bufio.Reader
	timeout time.Duration
}

func New(conn net.Conn, timeout time.Duration) *Conn {
	return &Conn{conn: conn, reader: bufio.NewReader(conn), timeout: timeout}
}

func (c *Conn) Write(payload []byte) error {
	if len(payload) > math.MaxUint16 {
		return fmt.Errorf("frame: paket çok büyük (%d bayt)", len(payload))
	}
	out := make([]byte, 0, len(payload)+6)
	out = append(out, header[:]...)
	out = binary.LittleEndian.AppendUint16(out, uint16(len(payload)))
	out = append(out, payload...)
	out = append(out, footer[:]...)

	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	_, err := c.conn.Write(out)
	return err
}

func (c *Conn) Read() ([]byte, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}

	var head [4]byte
	if _, err := io.ReadFull(c.reader, head[:]); err != nil {
		return nil, err
	}
	if head[0] != header[0] || head[1] != header[1] {
		return nil, ErrBadHeader
	}

	size := int(binary.LittleEndian.Uint16(head[2:]))
	rest := make([]byte, size+2)
	if _, err := io.ReadFull(c.reader, rest); err != nil {
		return nil, err
	}
	if rest[size] != footer[0] || rest[size+1] != footer[1] {
		return nil, ErrBadFooter
	}
	return rest[:size], nil
}
