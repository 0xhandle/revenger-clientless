package serverlist

import (
	"errors"
	"testing"

	"github.com/0xhandle/revenger-clientless/internal/packet"
)

type testEntry struct {
	id     int32
	name   string
	host   string
	port   uint16
	parent int32
}

func encode(entries ...testEntry) []byte {
	w := packet.NewWriter(Opcode).Int32(int32(len(entries)))
	for _, e := range entries {
		w.Int32(e.id).String(e.name).String(e.host).Uint16(e.port).Int32(e.parent)
	}
	return w.Bytes()
}

func TestParse(t *testing.T) {
	data := encode(
		testEntry{1000, "<color=white><b>REVENGER</b></color>", " ", 0, -1},
		testEntry{1004, "RevengerOnline - 1", "192.0.2.10", 19001, 1000},
		testEntry{2000, "<color=white><b>BERAMUS</b></color>", " ", 0, -1},
		testEntry{2004, "BERAMUS - 1", "192.0.2.11", 19002, 2000},
	)

	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []Server{
		{ID: 1004, Name: "RevengerOnline - 1", Group: "REVENGER", Host: "192.0.2.10", Port: 19001},
		{ID: 2004, Name: "BERAMUS - 1", Group: "BERAMUS", Host: "192.0.2.11", Port: 19002},
	}
	if len(got) != len(want) {
		t.Fatalf("%d sunucu geldi, beklenen %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sunucu %d = %+v, beklenen %+v", i, got[i], want[i])
		}
	}
}

func TestParseUnicodeName(t *testing.T) {
	got, err := Parse(encode(testEntry{1, "Türkiye Şehir", "192.0.2.1", 1, 7}))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "Türkiye Şehir" {
		t.Errorf("ad = %q", got[0].Name)
	}
}

func TestParseRejectsOtherOpcode(t *testing.T) {
	if _, err := Parse(packet.NewWriter(0x0010).Int32(0).Bytes()); !errors.Is(err, ErrUnexpectedOpcode) {
		t.Fatalf("hata = %v, beklenen %v", err, ErrUnexpectedOpcode)
	}
}

func TestParseTruncated(t *testing.T) {
	data := encode(testEntry{1, "A", "192.0.2.1", 1, 7})
	if _, err := Parse(data[:len(data)-3]); !errors.Is(err, packet.ErrShortPacket) {
		t.Fatalf("hata = %v, beklenen %v", err, packet.ErrShortPacket)
	}
}
