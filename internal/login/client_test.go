package login

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/0xhandle/revenger-clientless/internal/envelope"
	"github.com/0xhandle/revenger-clientless/internal/frame"
	"github.com/0xhandle/revenger-clientless/internal/packet"
	"github.com/0xhandle/revenger-clientless/internal/serverlist"
)

const (
	testUser     = "tester"
	testPassword = "secret"
	testPattern  = "1,2,3,6,9"
)

type fakeServer struct {
	askPattern bool
	listener   net.Listener
	failure    chan error
	pattern    chan string
}

func startFakeServer(t *testing.T, askPattern bool) *fakeServer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeServer{askPattern: askPattern, listener: l, failure: make(chan error, 1), pattern: make(chan string, 1)}
	go func() {
		conn, err := l.Accept()
		if err != nil {
			s.failure <- err
			return
		}
		defer conn.Close()
		if err := s.serve(frame.New(conn, 5*time.Second)); err != nil {
			s.failure <- err
		}
	}()
	t.Cleanup(func() { l.Close() })
	return s
}

func (s *fakeServer) serve(conn *frame.Conn) error {
	ping, err := conn.Read()
	if err != nil {
		return err
	}
	if op, _ := packet.Opcode(ping); op != opPing {
		return fmt.Errorf("beklenmeyen ping %x", ping)
	}
	if err := conn.Write(packet.NewWriter(opPing).Int32(1).Bytes()); err != nil {
		return err
	}

	if _, err := readSealed(conn, opVersion); err != nil {
		return err
	}
	if err := conn.Write(packet.NewWriter(0x0010).Int32(0).Bytes()); err != nil {
		return err
	}
	if err := conn.Write(packet.NewWriter(opVersion).String("3.5").Bytes()); err != nil {
		return err
	}

	request, err := readSealed(conn, opLogin)
	if err != nil {
		return err
	}
	r := packet.NewReader(request[2:])
	user, password := r.String(), r.String()
	if user != testUser || password != testPassword {
		return conn.Write(packet.NewWriter(opLogin).Int32(2).Bytes())
	}

	if err := conn.Write(packet.NewWriter(opLogin).Int32(loginSucceeded).String("token").Bytes()); err != nil {
		return err
	}

	if s.askPattern {
		if err := conn.Write(packet.NewWriter(opPattern).Int32(0).Bytes()); err != nil {
			return err
		}
		reply, err := readSealed(conn, opPattern)
		if err != nil {
			return err
		}
		r := packet.NewReader(reply[2:])
		r.Int32()
		s.pattern <- r.String()
	}

	list := packet.NewWriter(serverlist.Opcode).Int32(2).
		Int32(1000).String("<b>REVENGER</b>").String(" ").Uint16(0).Int32(-1).
		Int32(1004).String("RevengerOnline - 1").String("192.0.2.10").Uint16(19001).Int32(1000)
	return conn.Write(list.Bytes())
}

func readSealed(conn *frame.Conn, opcode uint16) ([]byte, error) {
	payload, err := conn.Read()
	if err != nil {
		return nil, err
	}
	inner, err := envelope.Open(payload)
	if err != nil {
		return nil, err
	}
	if op, _ := packet.Opcode(inner); op != opcode {
		return nil, fmt.Errorf("beklenen opcode %04x, gelen %04x", opcode, op)
	}
	return inner, nil
}

func fetch(t *testing.T, server *fakeServer, password, pattern string) ([]serverlist.Server, error) {
	t.Helper()
	servers, err := FetchServerList(context.Background(), Config{
		Address:  server.listener.Addr().String(),
		Username: testUser,
		Password: password,
		Pattern:  pattern,
		Device:   DefaultDevice("0123456789abcdef0123456789abcdef"),
		Timeout:  5 * time.Second,
	})
	select {
	case serverErr := <-server.failure:
		t.Fatalf("sahte sunucu: %v", serverErr)
	default:
	}
	return servers, err
}

func TestFetchServerList(t *testing.T) {
	servers, err := fetch(t, startFakeServer(t, false), testPassword, "")
	if err != nil {
		t.Fatal(err)
	}
	want := serverlist.Server{ID: 1004, Name: "RevengerOnline - 1", Group: "REVENGER", Host: "192.0.2.10", Port: 19001}
	if len(servers) != 1 || servers[0] != want {
		t.Fatalf("sunucular = %+v, beklenen %+v", servers, want)
	}
}

func TestFetchServerListWithPattern(t *testing.T) {
	server := startFakeServer(t, true)
	servers, err := fetch(t, server, testPassword, testPattern)
	if err != nil {
		t.Fatal(err)
	}
	if got := <-server.pattern; got != testPattern {
		t.Errorf("desen = %q, beklenen %q", got, testPattern)
	}
	if len(servers) != 1 {
		t.Errorf("%d sunucu geldi, beklenen 1", len(servers))
	}
}

func TestFetchServerListPatternRequired(t *testing.T) {
	_, err := fetch(t, startFakeServer(t, true), testPassword, "")
	if !errors.Is(err, ErrPatternRequired) {
		t.Fatalf("hata = %v, beklenen %v", err, ErrPatternRequired)
	}
}

func TestFetchServerListWrongPassword(t *testing.T) {
	_, err := fetch(t, startFakeServer(t, false), "yanlis", "")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("hata = %v, beklenen %v", err, ErrInvalidCredentials)
	}
}
