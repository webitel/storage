package apis

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	engine "github.com/webitel/engine/model"
)

type HttpRange struct {
	Start, Length int64
}

type ListResponse struct {
	Total int64       `json:"total"`
	Items interface{} `json:"items"`
}

func (r HttpRange) ContentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.Start, r.Start+r.Length-1, size)
}

var errFailedToOverlapRange = engine.NewBadRequestError("api.helper.parse_range.failed_to_overlap.app_error", "")
var errFailedRange = engine.NewBadRequestError("api.helper.parse_range.failed_range.app_error", "")

func parseRange(s string, size int64) ([]HttpRange, engine.AppError) {
	if s == "" {
		return nil, nil // header not present
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, errFailedRange
	}
	var ranges []HttpRange
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = strings.TrimSpace(ra)
		if ra == "" {
			continue
		}
		i := strings.Index(ra, "-")
		if i < 0 {
			return nil, errFailedRange
		}
		start, end := strings.TrimSpace(ra[:i]), strings.TrimSpace(ra[i+1:])
		var r HttpRange

		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file.
			i, err := strconv.ParseInt(end, 10, 64)
			if err != nil {
				return nil, errFailedRange
			}
			if i > size {
				i = size
			}
			r.Start = size - i
			r.Length = size - r.Start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, errFailedRange
			}
			if i >= size {
				// If the range begins after the size of the content,
				// then it does not overlap.
				noOverlap = true
				continue
			}
			r.Start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.Length = size - r.Start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.Start > i {
					return nil, errFailedRange
				}
				if i >= size {
					i = size - 1
				}
				r.Length = i - r.Start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		return nil, errFailedToOverlapRange
	}
	return ranges, nil
}

func (list *ListResponse) ToJson() string {
	b, _ := json.Marshal(list)
	return string(b)
}
