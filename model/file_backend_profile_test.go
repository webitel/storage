package model

import "testing"

func TestFileBackendProfileLocalType(t *testing.T) {
	local := func(dir string) *FileBackendProfile {
		return &FileBackendProfile{Type: FileDriverLocal, Properties: StringInterface{BackendProfileDirectoryField: dir}}
	}

	if err := local("/data").IsValidType(nil); err == nil {
		t.Error("create local: want an error")
	}
	if err := (&FileBackendProfile{Type: FileDriverS3}).IsValidType(nil); err != nil {
		t.Errorf("create s3: %v", err)
	}

	stored := local("/data")
	if err := local("/data").IsValidType(stored); err != nil {
		t.Errorf("update local, same directory: %v", err)
	}
	if err := local("/etc").IsValidType(stored); err == nil {
		t.Error("update local, new directory: want an error")
	}
	// The type of a stored profile is what counts: an update does not set it.
	if err := (&FileBackendProfile{Properties: StringInterface{BackendProfileDirectoryField: "/etc"}}).IsValidType(stored); err == nil {
		t.Error("update local without type, new directory: want an error")
	}
	s3 := &FileBackendProfile{Type: FileDriverS3, Properties: StringInterface{"bucket_name": "a"}}
	if err := (&FileBackendProfile{Type: FileDriverS3, Properties: StringInterface{"bucket_name": "b"}}).IsValidType(s3); err != nil {
		t.Errorf("update s3: %v", err)
	}
}

// The stored profile a patch is checked against must not change with the
// patch, even when the patch writes into the properties it has.
func TestFileBackendProfileClone(t *testing.T) {
	old := &FileBackendProfile{Type: FileDriverLocal, Properties: StringInterface{BackendProfileDirectoryField: "/data"}}
	stored := old.Clone()

	old.Properties[BackendProfileDirectoryField] = "/etc"
	if got := stored.Properties.GetString(BackendProfileDirectoryField); got != "/data" {
		t.Fatalf("stored directory = %q, want /data", got)
	}
	if err := old.IsValidType(stored); err == nil {
		t.Error("directory changed in place: want an error")
	}

	if (&FileBackendProfile{}).Clone().Properties != nil {
		t.Error("clone of nil properties is not nil")
	}
}
