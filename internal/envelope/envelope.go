package envelope

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math/big"
)

const (
	Opcode     = 0x79E0
	headerSize = 2 + 4 + 4

	minSeed = 100000
	maxSeed = 999999
)

var ErrMalformed = errors.New("envelope: bozuk şifreli paket")

func Seal(inner []byte) ([]byte, error) {
	seed, err := randomSeed()
	if err != nil {
		return nil, err
	}
	return SealWithSeed(inner, seed)
}

func SealWithSeed(inner []byte, seed uint32) ([]byte, error) {
	block, err := aes.NewCipher(key(seed))
	if err != nil {
		return nil, err
	}

	padded := make([]byte, (len(inner)+aes.BlockSize-1)/aes.BlockSize*aes.BlockSize)
	copy(padded, inner)
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(encrypted, padded)

	out := make([]byte, 0, headerSize+len(encrypted))
	out = binary.LittleEndian.AppendUint16(out, Opcode)
	out = binary.LittleEndian.AppendUint32(out, seed)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(encrypted)))
	return append(out, encrypted...), nil
}

func IsSealed(payload []byte) bool {
	return len(payload) >= 2 && binary.LittleEndian.Uint16(payload) == Opcode
}

func Open(payload []byte) ([]byte, error) {
	if len(payload) < headerSize || !IsSealed(payload) {
		return nil, ErrMalformed
	}
	seed := binary.LittleEndian.Uint32(payload[2:])
	size := int(binary.LittleEndian.Uint32(payload[6:]))
	if size%aes.BlockSize != 0 || headerSize+size > len(payload) {
		return nil, ErrMalformed
	}

	block, err := aes.NewCipher(key(seed))
	if err != nil {
		return nil, err
	}
	out := make([]byte, size)
	cipher.NewCBCDecrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(out, payload[headerSize:headerSize+size])
	return out, nil
}

func key(seed uint32) []byte {
	k := make([]byte, 16)
	binary.LittleEndian.PutUint32(k[12:], seed)
	return k
}

func randomSeed() (uint32, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(maxSeed-minSeed+1))
	if err != nil {
		return 0, err
	}
	return uint32(n.Int64()) + minSeed, nil
}
