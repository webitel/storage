package model

import "testing"

func TestSafeStoreName(t *testing.T) {
	for in, want := range map[string]string{
		"record.wav":                "record.wav",
		"a..b.txt":                  "a..b.txt",
		"abc_../../2/x.txt":         "abc_.._.._2_x.txt",
		"/etc/passwd":               "_etc_passwd",
		`..\..\x`:                   ".._.._x",
		".":                         "_.",
		"..":                        "_..",
		"":                          "",
		"uuid_name with spaces.mp3": "uuid_name with spaces.mp3",
	} {
		if got := SafeStoreName(in); got != want {
			t.Errorf("SafeStoreName(%q) = %q, want %q", in, got, want)
		}
	}

	j := &JobUploadFile{Uuid: "../../2", BaseFile: BaseFile{Name: "x.wav"}}
	if got := j.GetStoreName(); got != ".._.._2_x.wav" {
		t.Errorf("JobUploadFile.GetStoreName() = %q", got)
	}
}
