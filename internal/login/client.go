package login

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/0xhandle/revenger-clientless/internal/envelope"
	"github.com/0xhandle/revenger-clientless/internal/frame"
	"github.com/0xhandle/revenger-clientless/internal/packet"
	"github.com/0xhandle/revenger-clientless/internal/serverlist"
)

const (
	opLogin   = 0x0001
	opVersion = 0x0003
	opPattern = 0x0009
	opPing    = 0x01FE

	loginSucceeded = 1
)

var (
	ErrInvalidCredentials = errors.New("login: kullanıcı adı veya şifre hatalı")
	ErrPatternRequired    = errors.New("login: sunucu bu cihaz için desen istiyor, -pattern ile desen ver")
)

type Device struct {
	ID             string
	Model          string
	Name           string
	Graphics       string
	GraphicsMemory int32
	SystemMemory   int32
}

func DefaultDevice(id string) Device {
	return Device{
		ID:             id,
		Model:          "Pixel 4",
		Name:           "google Pixel 4",
		Graphics:       "Vulkan google Pixel 4 False",
		GraphicsMemory: 2048,
		SystemMemory:   4096,
	}
}

type Config struct {
	Address  string
	Username string
	Password string
	Pattern  string
	Device   Device
	Timeout  time.Duration
}

func FetchServerList(ctx context.Context, cfg Config) ([]serverlist.Server, error) {
	dialer := net.Dialer{Timeout: cfg.Timeout}
	raw, err := dialer.DialContext(ctx, "tcp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("login: bağlantı kurulamadı: %w", err)
	}
	defer raw.Close()
	stop := context.AfterFunc(ctx, func() { raw.Close() })
	defer stop()

	conn := frame.New(raw, cfg.Timeout)

	if err := conn.Write(packet.NewWriter(opPing).Int32(0).Bytes()); err != nil {
		return nil, fmt.Errorf("login: ping gönderilemedi: %w", err)
	}
	if _, err := waitFor(conn, opPing); err != nil {
		return nil, fmt.Errorf("login: ping cevabı alınamadı: %w", err)
	}

	if err := writeSealed(conn, packet.NewWriter(opVersion).Uint16(1).Bytes()); err != nil {
		return nil, fmt.Errorf("login: versiyon isteği gönderilemedi: %w", err)
	}
	if _, err := waitFor(conn, opVersion); err != nil {
		return nil, fmt.Errorf("login: versiyon cevabı alınamadı: %w", err)
	}

	if err := writeSealed(conn, loginRequest(cfg)); err != nil {
		return nil, fmt.Errorf("login: giriş paketi gönderilemedi: %w", err)
	}
	return awaitServerList(conn, cfg.Pattern)
}

func awaitServerList(conn *frame.Conn, pattern string) ([]serverlist.Server, error) {
	for {
		message, err := readMessage(conn)
		if err != nil {
			return nil, fmt.Errorf("login: sunucu listesi alınamadı: %w", err)
		}
		op, _ := packet.Opcode(message)

		switch op {
		case opLogin:
			r := packet.NewReader(message[2:])
			if result := r.Int32(); result != loginSucceeded {
				return nil, fmt.Errorf("%w (kod %d)", ErrInvalidCredentials, result)
			}
		case opPattern:
			if pattern == "" {
				return nil, ErrPatternRequired
			}
			step := packet.NewReader(message[2:]).Int32()
			if err := writeSealed(conn, packet.NewWriter(opPattern).Int32(step).String(pattern).Bytes()); err != nil {
				return nil, fmt.Errorf("login: desen gönderilemedi: %w", err)
			}
		case serverlist.Opcode:
			return serverlist.Parse(message)
		}
	}
}

func loginRequest(cfg Config) []byte {
	d := cfg.Device
	return packet.NewWriter(opLogin).
		String(cfg.Username).
		String(cfg.Password).
		String(d.ID).
		String(d.Model).
		String(d.Name).
		String(d.Graphics).
		Int32(d.GraphicsMemory).
		Int32(d.SystemMemory).
		String(d.ID).
		Bytes()
}

func writeSealed(conn *frame.Conn, inner []byte) error {
	sealed, err := envelope.Seal(inner)
	if err != nil {
		return err
	}
	return conn.Write(sealed)
}

func readMessage(conn *frame.Conn) ([]byte, error) {
	for {
		payload, err := conn.Read()
		if err != nil {
			return nil, err
		}
		if envelope.IsSealed(payload) {
			payload, err = envelope.Open(payload)
			if err != nil {
				return nil, err
			}
		}
		if len(payload) >= 2 {
			return payload, nil
		}
	}
}

func waitFor(conn *frame.Conn, opcode uint16) ([]byte, error) {
	for {
		message, err := readMessage(conn)
		if err != nil {
			return nil, err
		}
		if op, _ := packet.Opcode(message); op == opcode {
			return message, nil
		}
	}
}
