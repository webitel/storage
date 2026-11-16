package app

import (
	"errors"
	"testing"

	"github.com/webitel/storage/model"
)

func TestPolicyReader_checkMimeType(t *testing.T) {
	t.Parallel()

	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	textBytes := []byte("Hello, World! This is a plain text file.")
	wavBytes := []byte("RIFF____WAVEfmt ")
	webpBytes := []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")
	binaryBytes := []byte{0x00, 0x01, 0x02, 0x03, 0xFA, 0xFB, 0xFC, 0xFD}
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	tests := []struct {
		name         string
		declaredMime string
		inputBytes   []byte
		expectedErr  error
	}{
		{
			name:         "Exact match JPEG",
			declaredMime: "image/jpeg",
			inputBytes:   jpegBytes,
			expectedErr:  nil,
		},
		{
			name:         "Prefix match with parameters (text/plain with charset)",
			declaredMime: "text/plain; charset=utf-8",
			inputBytes:   textBytes,
			expectedErr:  nil,
		},
		{
			name:         "Special rule for audio/wav",
			declaredMime: "audio/wav",
			inputBytes:   wavBytes,
			expectedErr:  nil,
		},
		{
			name:         "Unknown type (empty bytes / generic)",
			declaredMime: "image/jpeg",
			inputBytes:   []byte{},
			expectedErr:  model.PolicyErrorExtUnknown,
		},
		{
			name:         "Suspicious mismatch (Declared PNG, actual JPEG)",
			declaredMime: "image/png",
			inputBytes:   jpegBytes,
			expectedErr:  model.PolicyErrorExtSuspicious,
		},
		{
			name:         "Nil slice input",
			declaredMime: "image/jpeg",
			inputBytes:   nil,
			expectedErr:  model.PolicyErrorExtUnknown,
		},
		{
			name:         "Unknown binary stream (application/octet-stream)",
			declaredMime: "application/octet-stream",
			inputBytes:   binaryBytes,
			expectedErr:  model.PolicyErrorExtUnknown,
		},
		{
			name:         "RIFF container spoofing (Declared WAV, actual WebP)",
			declaredMime: "audio/wav",
			inputBytes:   webpBytes,
			expectedErr:  model.PolicyErrorExtSuspicious,
		},
		{
			name:         "Empty declared MimeType with valid file",
			declaredMime: "",
			inputBytes:   pngBytes,
			expectedErr:  model.PolicyErrorExtSuspicious,
		},
		{
			name:         "Uppercase declared MimeType (RFC 2045 compliance check)",
			declaredMime: "IMAGE/JPEG",
			inputBytes:   jpegBytes,
			expectedErr:  model.PolicyErrorExtSuspicious,
		},
		{
			name:         "Leading/trailing spaces in declared MimeType",
			declaredMime: " image/png ",
			inputBytes:   pngBytes,
			expectedErr:  model.PolicyErrorExtSuspicious,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &PolicyReader{
				f: &model.BaseFile{
					MimeType: tt.declaredMime,
				},
			}

			err := r.checkMimeType(tt.inputBytes)

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("checkMimeType() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
