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
			name:         "exact_match_jpeg",
			declaredMime: "image/jpeg",
			inputBytes:   jpegBytes,
			expectedErr:  nil,
		},
		{
			name:         "prefix_match_with_parameters_(text/plain_with_charset)",
			declaredMime: "text/plain; charset=utf-8",
			inputBytes:   textBytes,
			expectedErr:  nil,
		},
		{
			name:         "special_rule_for_audio/wav",
			declaredMime: "audio/wav",
			inputBytes:   wavBytes,
			expectedErr:  nil,
		},
		{
			name:         "unknown_type_(empty_bytes_/_generic)",
			declaredMime: "image/jpeg",
			inputBytes:   []byte{},
			expectedErr:  model.PolicyErrorExtUnknown,
		},
		{
			name:         "suspicious_mismatch_(declared_png,_actual_jpeg)",
			declaredMime: "image/png",
			inputBytes:   jpegBytes,
			expectedErr:  model.PolicyErrorExtSuspicious,
		},
		{
			name:         "nil_slice_input",
			declaredMime: "image/jpeg",
			inputBytes:   nil,
			expectedErr:  model.PolicyErrorExtUnknown,
		},
		{
			name:         "unknown_binary_stream_(application/octet_stream)",
			declaredMime: "application/octet-stream",
			inputBytes:   binaryBytes,
			expectedErr:  model.PolicyErrorExtUnknown,
		},
		{
			name:         "riff_container_spoofing_(declared_wav,_actual_web_p)",
			declaredMime: "audio/wav",
			inputBytes:   webpBytes,
			expectedErr:  model.PolicyErrorExtSuspicious,
		},
		{
			name:         "empty_declared_mime_type_with_valid_file",
			declaredMime: "",
			inputBytes:   pngBytes,
			expectedErr:  model.PolicyErrorExtSuspicious,
		},
		{
			name:         "uppercase_declared_mime_type_(rfc_2045_compliance_check)",
			declaredMime: "IMAGE/JPEG",
			inputBytes:   jpegBytes,
			expectedErr:  nil,
		},
		{
			name:         "leading/trailing_spaces_in_declared_mime_type",
			declaredMime: " image/png ",
			inputBytes:   pngBytes,
			expectedErr:  nil,
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
