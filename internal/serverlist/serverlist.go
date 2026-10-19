package serverlist

import (
	"errors"
	"strings"

	"github.com/0xhandle/revenger-clientless/internal/packet"
)

const (
	Opcode      = 0x0002
	groupParent = -1
)

var ErrUnexpectedOpcode = errors.New("serverlist: beklenmeyen opcode")

type Server struct {
	ID    int32  `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
	Host  string `json:"host"`
	Port  uint16 `json:"port"`
}

type entry struct {
	id     int32
	name   string
	host   string
	port   uint16
	parent int32
}

func Parse(payload []byte) ([]Server, error) {
	r := packet.NewReader(payload)
	if r.Uint16() != Opcode {
		if err := r.Err(); err != nil {
			return nil, err
		}
		return nil, ErrUnexpectedOpcode
	}

	count := int(r.Int32())
	entries := make([]entry, 0, max(count, 0))
	for range count {
		e := entry{
			id:   r.Int32(),
			name: r.String(),
			host: r.String(),
			port: r.Uint16(),
		}
		e.parent = r.Int32()
		if err := r.Err(); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	groups := make(map[int32]string)
	for _, e := range entries {
		if e.parent == groupParent {
			groups[e.id] = stripTags(e.name)
		}
	}

	servers := make([]Server, 0, len(entries))
	for _, e := range entries {
		if e.parent == groupParent {
			continue
		}
		servers = append(servers, Server{
			ID:    e.id,
			Name:  e.name,
			Group: groups[e.parent],
			Host:  e.host,
			Port:  e.port,
		})
	}
	return servers, nil
}

func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
