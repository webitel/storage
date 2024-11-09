package model

const (
	PERMISSION_SCOPE_BACKEND_PROFILE = "storage_profile"
	PERMISSION_SCOPE_MEDIA_FILE      = "media_file"
	PERMISSION_SCOPE_RECORD_FILE     = "record_file"

	PermissionScopeCognitiveProfile = "cognitive_profile"
	PermissionScopeImportTemplate   = "import_template"
	PermissionScopeFilePolicy       = "storage_profile" //"file_policies"
	PermissionScopeFiles            = "files"           //"file_policies"
)

const (
	PermissionActionAccessCallRecordings = "playback_record_file"
)
