package packet

import (
	"encoding/binary"
	"errors"
	"unicode/utf16"
)

var ErrShortPacket = errors.New("packet: veri beklenenden kısa")

type Writer struct {
	buf []byte
}

func NewWriter(opcode uint16) *Writer {
	return (&Writer{}).Uint16(opcode)
}

func (w *Writer) Uint16(v uint16) *Writer {
	w.buf = binary.LittleEndian.AppendUint16(w.buf, v)
	return w
}

func (w *Writer) Int32(v int32) *Writer {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, uint32(v))
	return w
}

func (w *Writer) String(s string) *Writer {
	units := utf16.Encode([]rune(s))
	w.Uint16(uint16(len(units)))
	for _, u := range units {
		w.Uint16(u)
	}
	return w
}

func (w *Writer) Bytes() []byte {
	return w.buf
}

type Reader struct {
	buf []byte
	err error
}

func NewReader(b []byte) *Reader {
	return &Reader{buf: b}
}

func (r *Reader) Err() error {
	return r.err
}

func (r *Reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || len(r.buf) < n {
		r.err = ErrShortPacket
		return nil
	}
	out := r.buf[:n]
	r.buf = r.buf[n:]
	return out
}

func (r *Reader) Uint16() uint16 {
	b := r.take(2)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

func (r *Reader) Int32() int32 {
	b := r.take(4)
	if b == nil {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(b))
}

func (r *Reader) String() string {
	n := int(r.Uint16())
	b := r.take(n * 2)
	if b == nil {
		return ""
	}
	units := make([]uint16, n)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(units))
}

func Opcode(payload []byte) (uint16, bool) {
	if len(payload) < 2 {
		return 0, false
	}
	return binary.LittleEndian.Uint16(payload), true
}
