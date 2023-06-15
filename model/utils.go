package model

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	goi18n "github.com/nicksnyder/go-i18n/i18n"
	"github.com/pborman/uuid"
)

type StringInterface map[string]interface{}
type StringMap map[string]string
type StringArray []string
type JSON string

type Lookup struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func (l *Lookup) GetSafeId() *int {
	if l == nil || l.Id == 0 {

		return nil
	}

	return &l.Id
}

func (s *StringInterface) ToJson() string {
	if s == nil {
		return ""
	}
	b, _ := json.Marshal(s)
	return string(b)
}

func (s StringInterface) GetString(name string) string {
	if v, ok := s[name]; ok {
		return fmt.Sprintf("%s", v)
	}
	return ""
}
func (s StringInterface) GetBool(name string) bool {
	if v, ok := s[name].(bool); ok {
		return v
	}
	return false
}

type AppError struct {
	Id            string `json:"id"`
	Message       string `json:"message"`               // Message to be display to the end user without debugging information
	DetailedError string `json:"detailed_error"`        // Internal error string to help the developer
	RequestId     string `json:"request_id,omitempty"`  // The RequestId that's also set in the header
	StatusCode    int    `json:"status_code,omitempty"` // The http status code
	Where         string `json:"-"`                     // The function where it happened in the form of Struct.Func
	IsOAuth       bool   `json:"is_oauth,omitempty"`    // Whether the error is OAuth specific
	params        map[string]interface{}
}

func (er *AppError) Error() string {
	return er.Where + ": " + er.Message + ", " + er.DetailedError
}

func NewAppError(where string, id string, params map[string]interface{}, details string, status int) *AppError {
	ap := &AppError{}
	ap.Id = id
	ap.params = params
	ap.Message = id
	ap.Where = where
	ap.DetailedError = details
	ap.StatusCode = status
	ap.IsOAuth = false
	ap.Translate(translateFunc)
	return ap
}

func (er *AppError) SystemMessage(T goi18n.TranslateFunc) string {
	if er.params == nil {
		return T(er.Id)
	} else {
		return T(er.Id, er.params)
	}
}

func (er *AppError) ToJson() string {
	b, _ := json.Marshal(er)
	return string(b)
}

func (er *AppError) Text() string {
	if er.DetailedError != "" {
		return er.DetailedError
	}
	return er.Message
}

func (er *AppError) Translate(T goi18n.TranslateFunc) {
	if T == nil {
		er.Message = er.Id
		return
	}

	if er.params == nil {
		er.Message = T(er.Id)
	} else {
		er.Message = T(er.Id, er.params)
	}
}

var translateFunc goi18n.TranslateFunc = nil

func AppErrorInit(t goi18n.TranslateFunc) {
	translateFunc = t
}

var encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769")

// NewId is a globally unique identifier.  It is a [A-Z0-9] string 26
// characters long.  It is a UUID version 4 Guid that is zbased32 encoded
// with the padding stripped off.
func NewId() string {
	var b bytes.Buffer
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	b.Truncate(26) // removes the '==' padding
	return b.String()
}

// MapToJson converts a map to a json string
func MapToJson(objmap map[string]string) string {
	b, _ := json.Marshal(objmap)
	return string(b)
}

// MapFromJson will decode the key/value pair map
func MapFromJson(data io.Reader) map[string]string {
	decoder := json.NewDecoder(data)

	var objmap map[string]string
	if err := decoder.Decode(&objmap); err != nil {
		return make(map[string]string)
	} else {
		return objmap
	}
}

func ArrayToJson(objmap []string) string {
	b, _ := json.Marshal(objmap)
	return string(b)
}

func ArrayFromJson(data io.Reader) []string {
	decoder := json.NewDecoder(data)

	var objmap []string
	if err := decoder.Decode(&objmap); err != nil {
		return make([]string, 0)
	} else {
		return objmap
	}
}

func StringInterfaceToJson(objmap map[string]interface{}) string {
	b, _ := json.Marshal(objmap)
	return string(b)
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func EncodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	return r
}
