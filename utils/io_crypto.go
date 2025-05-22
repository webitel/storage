package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"io"
	"os"
)

const (
	BlockSize          = 256 * 1024 // 256 KB
	NonceSize          = chacha20poly1305.NonceSize
	TagSize            = chacha20poly1305.Overhead
	EncryptedBlockSize = NonceSize + BlockSize + TagSize
)

type Chipher cipher.AEAD

func NewChipher(keyFile string) (Chipher, error) {
	sharedSecret, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	hkdfReader := hkdf.New(sha256.New, sharedSecret, nil, nil)
	encryptionKey := make([]byte, chacha20poly1305.KeySize)
	if _, err = io.ReadFull(hkdfReader, encryptionKey); err != nil {
		return nil, err
	}

	return chacha20poly1305.New(encryptionKey)
}

func NewDecryptingReader(src io.ReadCloser, aead Chipher, innerOffset int64) io.ReadCloser {
	return &decryptingReader{
		src:         src,
		aead:        aead,
		innerOffset: innerOffset,
		nonce:       make([]byte, NonceSize),
		body:        make([]byte, BlockSize+TagSize),
		dstBuf:      make([]byte, 0, BlockSize),
	}
}

func NewEncryptingReader(src io.Reader, aead Chipher) io.Reader {
	return &encryptingReader{
		src:       src,
		aead:      aead,
		nonce:     make([]byte, NonceSize),
		body:      make([]byte, BlockSize),
		cipherBuf: make([]byte, 0, BlockSize+TagSize),
		buf:       make([]byte, 0, NonceSize+BlockSize+TagSize),
	}
}

func EstimateOriginalSize(encSize int64) (int64, error) {
	numFullBlocks := encSize / int64(EncryptedBlockSize)
	remaining := encSize % int64(EncryptedBlockSize)

	if remaining > 0 && remaining < NonceSize+TagSize {
		return 0, fmt.Errorf("invalid remaining block: too small to be valid")
	}

	original := numFullBlocks * BlockSize
	if remaining > 0 {
		original += remaining - NonceSize - TagSize
	}

	return original, nil
}

func EstimateFirstBlockOffset(file File, offset int64) int64 {
	if file.IsEncrypted() {
		offset = (offset / BlockSize) * EncryptedBlockSize
	}
	return offset
}

type encryptingReader struct {
	src       io.Reader
	aead      cipher.AEAD
	nonce     []byte
	buf       []byte
	body      []byte
	cipherBuf []byte
	offset    int
	err       error
}
type decryptingReader struct {
	src         io.ReadCloser
	aead        cipher.AEAD
	buf         []byte
	offset      int
	err         error
	nonce       []byte
	body        []byte
	dstBuf      []byte
	innerOffset int64
}

func (er *decryptingReader) Close() error {
	return er.src.Close()
}

func (er *encryptingReader) Read(p []byte) (int, error) {
	if er.offset >= len(er.buf) && er.err == nil {
		n, err := io.ReadFull(er.src, er.body)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			er.err = err
			return 0, err
		}

		if _, err = rand.Read(er.nonce); err != nil {
			return 0, err
		}

		cipherBody := er.aead.Seal(er.cipherBuf[:0], er.nonce, er.body[:n], nil)
		er.buf = er.buf[:0]
		er.buf = append(er.buf, er.nonce...)
		er.buf = append(er.buf, cipherBody...)

		er.offset = 0

		if errors.Is(err, io.ErrUnexpectedEOF) || n == 0 {
			er.err = io.EOF
		}
	}

	if er.offset >= len(er.buf) {
		return 0, er.err
	}

	n := copy(p, er.buf[er.offset:])
	er.offset += n
	return n, nil
}

func (dr *decryptingReader) Read(p []byte) (int, error) {
	if dr.offset >= len(dr.buf) && dr.err == nil {
		var plainBody []byte

		n, err := io.ReadFull(dr.src, dr.nonce)
		if err != nil {
			dr.err = err
			return 0, err
		}

		n, err = io.ReadFull(dr.src, dr.body)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			dr.err = err
			return 0, err
		}

		plainBody, err = dr.aead.Open(dr.dstBuf[:0], dr.nonce, dr.body[:n], nil)
		if err != nil {
			dr.err = err
			return 0, err
		}

		dr.dstBuf = plainBody

		if dr.innerOffset != 0 {
			plainBody = plainBody[(dr.innerOffset % BlockSize):]
			//
			dr.innerOffset = 0
		}

		dr.buf = plainBody
		dr.offset = 0

		if errors.Is(err, io.ErrUnexpectedEOF) {
			dr.err = io.EOF
		}
	}

	if dr.offset >= len(dr.buf) {
		return 0, dr.err
	}

	n := copy(p, dr.buf[dr.offset:])
	dr.offset += n
	return n, nil
}
