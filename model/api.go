package model

const (
	API_URL_SUFFIX_V2          = "/api/v2"
	API_URL_SUFFIX_V3          = "/api/v3"
	API_INTERNAL_URL_SUFFIX_V1 = "/sys"
	API_URL_SUFFIX             = API_URL_SUFFIX_V2

	HEADER_REQUEST_ID         = "X-Request-ID"
	HEADER_TOKEN              = "X-Webitel-Access"
	HEADER_BEARER             = "BEARER"
	HEADER_AUTH               = "Authorization"
	HEADER_FORWARDED          = "X-Forwarded-For"
	HEADER_REAL_IP            = "X-Real-IP"
	HEADER_REQUESTED_WITH     = "X-Requested-With"
	HEADER_REQUESTED_WITH_XML = "XMLHttpRequest"

	STATUS    = "status"
	STATUS_OK = "OK"
)

const (
	AnyFileRouteName = "/any/file"
)

const PreSignRequireChannelParam = "require_channel"
