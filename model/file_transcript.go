package model

import (
	"encoding/json"
	"time"
)

type TranscriptRange struct {
	StartSec float64 `json:"start_sec" db:"start_sec"`
	EndSec   float64 `json:"end_sec" db:"end_sec" `
}

type TranscriptWord struct {
	Word string `json:"word"`
	TranscriptRange
}

type TranscriptPhrase struct {
	TranscriptRange
	Channel uint32           `json:"channel" db:"channel"`
	Itn     string           `json:"itn" db:"-"`
	Display string           `json:"display" db:"phrase"`
	Lexical string           `json:"lexical" db:"-"`
	Words   []TranscriptWord `json:"words" db:"-"`
}

type TranscriptChannel struct {
	Channel int    `json:"channel" db:"channel"`
	Display string `json:"display" db:"display"`
	Lexical string `json:"lexical" db:"lexical"`
}

type FileTranscript struct {
	Id         int64               `json:"id" db:"id"`
	File       Lookup              `json:"file" db:"file"`
	Profile    *Lookup             `json:"profile" db:"profile"`
	Transcript string              `json:"transcript" db:"transcript"`
	Log        json.RawMessage     `json:"log" db:"log"`
	CreatedAt  time.Time           `json:"created_at" db:"created_at"`
	Locale     string              `json:"locale" db:"locale"`
	Phrases    []TranscriptPhrase  `json:"phrases" db:"phrases"`
	Channels   []TranscriptChannel `json:"channels" db:"channels"`
}

type FileTranscriptJob struct {
	Id        int64 `json:"id" db:"id"`
	FileId    int64 `json:"file_id" db:"file_id"`
	CreatedAt int64 `json:"created_at" db:"created_at"`
	State     uint8 `json:"state" db:"state"`
}

func (f *FileTranscript) TidyTranscript() string {
	t := ""

	for k, v := range f.Channels {
		if k > 0 {
			t += " "
		}

		t += v.Display
	}

	return t
}

func (f *FileTranscript) JsonPhrases() []byte {
	if f.Phrases != nil {
		d, _ := json.Marshal(f.Phrases)
		return d
	}

	return nil
}

func (f *FileTranscript) JsonChannels() []byte {
	if f.Channels != nil {
		d, _ := json.Marshal(f.Channels)
		return d
	}

	return nil
}
