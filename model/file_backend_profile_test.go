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

// The client does not get access_key back, so a patch without it (or with it
// empty) keeps the stored one (WTEL-4344).
func TestFileBackendProfilePatchAccessKey(t *testing.T) {
	stored := func() *FileBackendProfile {
		return &FileBackendProfile{Properties: StringInterface{BackendProfileAccessKeyField: "secret", "bucket_name": "b"}}
	}

	f := stored()
	props := StringInterface{"bucket_name": "b2"}
	f.Patch(&FileBackendProfilePath{Properties: &props})
	if got := f.Properties.GetString(BackendProfileAccessKeyField); got != "secret" {
		t.Errorf("no key: access_key = %q, want secret", got)
	}
	if got := f.Properties.GetString("bucket_name"); got != "b2" {
		t.Errorf("bucket_name = %q, want b2", got)
	}

	f = stored()
	props = StringInterface{BackendProfileAccessKeyField: "new"}
	f.Patch(&FileBackendProfilePath{Properties: &props})
	if got := f.Properties.GetString(BackendProfileAccessKeyField); got != "new" {
		t.Errorf("new key: access_key = %q, want new", got)
	}

	f = stored()
	var null StringInterface
	f.Patch(&FileBackendProfilePath{Properties: &null})
	if got := f.Properties.GetString(BackendProfileAccessKeyField); got != "secret" {
		t.Errorf("null properties: access_key = %q, want secret", got)
	}

	f = stored()
	priority := 5
	f.Patch(&FileBackendProfilePath{Priority: &priority})
	if f.Priority != 5 || f.Properties.GetString(BackendProfileAccessKeyField) != "secret" {
		t.Errorf("priority only: priority = %d, properties = %v", f.Priority, f.Properties)
	}

	f = &FileBackendProfile{}
	props = StringInterface{"bucket_name": "b"}
	f.Patch(&FileBackendProfilePath{Properties: &props})
	if _, ok := f.Properties[BackendProfileAccessKeyField]; ok {
		t.Errorf("no stored key: properties = %v, want no access_key", f.Properties)
	}
}
