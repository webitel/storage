package apis

import (
	"net/url"
	"testing"

	"github.com/webitel/storage/model"
)

func TestUseThumbnail(t *testing.T) {
	thumb := &model.Thumbnail{BaseFile: model.BaseFile{Name: "thumbnail_v.mp4.png", MimeType: "image/png", Size: 10}}

	cases := []struct {
		name     string
		thumb    *model.Thumbnail
		query    url.Values
		wantMime string
	}{
		{"flag set serves thumbnail", thumb, url.Values{"fetch_thumbnail": {"true"}}, "image/png"},
		{"no flag serves file", thumb, url.Values{}, "video/mp4"},
		{"flag without thumbnail serves file", nil, url.Values{"fetch_thumbnail": {"true"}}, "video/mp4"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &model.File{BaseFile: model.BaseFile{Name: "v.mp4", MimeType: "video/mp4", Size: 1000}, Thumbnail: tc.thumb}

			useThumbnail(f, tc.query)

			if f.MimeType != tc.wantMime {
				t.Errorf("mime = %q, want %q", f.MimeType, tc.wantMime)
			}
		})
	}
}
