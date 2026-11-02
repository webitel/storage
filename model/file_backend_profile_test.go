package model

import "testing"

func TestFileBackendProfilePathPattern(t *testing.T) {
	profile := func(pattern string) *FileBackendProfile {
		return &FileBackendProfile{Properties: StringInterface{BackendProfilePathPatternField: pattern}}
	}

	for _, pattern := range []string{"", "$DOMAIN", "$DOMAIN/$Y/$M/$D/$H", "$DOMAIN/$CHANNEL/$MIME", "$Y/$m"} {
		if err := profile(pattern).IsValidPathPattern(nil); err != nil {
			t.Errorf("%q: %v", pattern, err)
		}
	}

	for _, pattern := range []string{
		"../2/$Y", "2/$Y", "/etc", "$DOMAIN/..", "$DOMAIN/../2", "$DOMAIN//$Y", "$DOMAIN/",
		"/$DOMAIN", "$DOMAIN-$Y", "records/$DOMAIN", "$DOMAIN\\..\\2", "$domain",
	} {
		if err := profile(pattern).IsValidPathPattern(nil); err == nil {
			t.Errorf("%q: want an error", pattern)
		}
	}

	// A stored profile keeps the pattern it has; a new one is checked.
	stored := profile("records/$Y")
	if err := profile("records/$Y").IsValidPathPattern(stored); err != nil {
		t.Errorf("unchanged pattern: %v", err)
	}
	if err := profile("../2/$Y").IsValidPathPattern(stored); err == nil {
		t.Error("changed pattern: want an error")
	}
	if err := (&FileBackendProfile{}).IsValidPathPattern(stored); err != nil {
		t.Errorf("pattern removed: %v", err)
	}
}
