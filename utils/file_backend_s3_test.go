package utils

import "testing"

// An upload's store name is "<uuid>_<id>_" + the client's file name, so
// "../../etc/cron.d/x" must not move the key out of the directory prefix.
func TestS3ObjectKeyPathEscape(t *testing.T) {
	for _, tc := range []struct{ dir, name string }{
		{"1/2026/10/30", "9_abc_/../../../../etc/cron.d/zz"},
		{"1/2026/10/30", "9_abc_/../../tmp/x"},
		{"1/2026/10/30", ".."},
		{"1/../../x", "9_abc_ok.txt"},
		{"", "../x"},
	} {
		if _, err := objectKey(tc.dir, tc.name); err == nil || err.GetId() != ErrPathEscapeId {
			t.Errorf("objectKey(%q, %q): err = %v, want %s", tc.dir, tc.name, err, ErrPathEscapeId)
		}
	}

	for _, tc := range []struct{ dir, name, want string }{
		{"1/2026/10/30", "9_abc_ok.txt", "1/2026/10/30/9_abc_ok.txt"},
		{"1/2026/10/30", "9_abc_a..b.txt", "1/2026/10/30/9_abc_a..b.txt"},
		{"1/2026/10/30", "9_abc_sub/x.txt", "1/2026/10/30/9_abc_sub/x.txt"},
	} {
		key, err := objectKey(tc.dir, tc.name)
		if err != nil || key != tc.want {
			t.Errorf("objectKey(%q, %q) = %q, %v; want %q", tc.dir, tc.name, key, err, tc.want)
		}
	}
}
