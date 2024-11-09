package model

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

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

func (s StringInterface) Remove(name string) {
	delete(s, name)
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

func TimeToInt64(t *time.Time) int64 {
	if t == nil {
		return 0
	}

	return t.UnixNano() / int64(time.Millisecond)
}
