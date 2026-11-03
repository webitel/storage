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
