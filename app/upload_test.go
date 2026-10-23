package app

import (
	"maps"
	"testing"

	"github.com/webitel/storage/model"
)

func TestThumbnailJobDoesNotShareProperties(t *testing.T) {
	tests := []struct {
		name  string
		props model.StringInterface
		want  model.StringInterface
	}{
		{
			name:  "no properties",
			props: nil,
			want:  nil,
		},
		{
			name:  "s3 location",
			props: model.StringInterface{"policy_id": 7, "encrypted": true, "location": "1/video/webm/record.webm"},
			want:  model.StringInterface{"policy_id": 7, "encrypted": true},
		},
		{
			name:  "local directory",
			props: model.StringInterface{"policy_id": 7, "directory": "1/video/webm"},
			want:  model.StringInterface{"policy_id": 7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &model.JobUploadFile{}
			file.Name = "record.webm"
			file.MimeType = "video/webm"
			file.Properties = maps.Clone(tt.props)

			thumbnailFile := thumbnailJob(file)

			if !maps.Equal(thumbnailFile.Properties, tt.want) {
				t.Fatalf("thumbnail properties = %v; want %v", thumbnailFile.Properties, tt.want)
			}

			// The backend writes the stored path of the thumbnail after the file.
			thumbnailFile.SetPropertyString("location", "1/image/png/thumbnail_record.webm.png")

			if !maps.Equal(file.Properties, tt.props) {
				t.Fatalf("file properties = %v after thumbnail upload; want %v", file.Properties, tt.props)
			}
		})
	}
}
