package envelope

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	for _, size := range []int{0, 1, 15, 16, 17, 200} {
		inner := bytes.Repeat([]byte{0x5A}, size)
		sealed, err := Seal(inner)
		if err != nil {
			t.Fatal(err)
		}
		if !IsSealed(sealed) {
			t.Fatalf("boyut %d: zarf opcode'u yok", size)
		}

		seed := binary.LittleEndian.Uint32(sealed[2:])
		if seed < minSeed || seed > maxSeed {
			t.Errorf("seed %d aralık dışında", seed)
		}

		opened, err := Open(sealed)
		if err != nil {
			t.Fatalf("boyut %d: %v", size, err)
		}
		if len(opened)%16 != 0 || !bytes.Equal(opened[:size], inner) || !bytes.Equal(opened[size:], make([]byte, len(opened)-size)) {
			t.Errorf("boyut %d: açılan veri uyuşmuyor: %x", size, opened)
		}
	}
}

func TestSealWithSeedIsDeterministic(t *testing.T) {
	a, _ := SealWithSeed([]byte{0x01, 0x00}, 123456)
	b, _ := SealWithSeed([]byte{0x01, 0x00}, 123456)
	if !bytes.Equal(a, b) {
		t.Fatal("aynı seed farklı çıktı üretti")
	}
	if got := binary.LittleEndian.Uint32(a[6:]); got != 16 {
		t.Errorf("şifreli uzunluk = %d, beklenen 16", got)
	}
}

func TestOpenRejectsMalformed(t *testing.T) {
	sealed, _ := SealWithSeed([]byte{0x01}, 111111)
	for name, payload := range map[string][]byte{
		"kısa":          sealed[:5],
		"eksik veri":    sealed[:len(sealed)-1],
		"farklı opcode": append([]byte{0x01, 0x00}, sealed[2:]...),
	} {
		if _, err := Open(payload); !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: hata = %v, beklenen %v", name, err, ErrMalformed)
		}
	}
}
