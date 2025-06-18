package apis

import (
	"encoding/json"
	"github.com/pion/webrtc/v4"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
	"io"
	"net/http"
)

type RequestStoreScreen struct {
	webrtc.SessionDescription
	Name       string             `json:"name"`
	Uuid       string             `json:"uuid"`
	ICEServers []webrtc.ICEServer `json:"ICEServers"`
}

func (api *API) InitWebRTC() {
	api.PublicRoutes.WebRTC.Handle("/upload/video", api.ApiSessionRequired(webrtcUploadVideo)).Methods("POST")
}

func webrtcUploadVideo(c *Context, w http.ResponseWriter, r *http.Request) {
	var offer RequestStoreScreen
	var answer *webrtc.SessionDescription

	data, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	defer r.Body.Close()

	err = decodeSdp(data, &offer)
	if err != nil {
		panic(err.Error())
	}

	name := offer.Name
	if name == "" {
		name = model.NewId()
	}

	file := &model.JobUploadFile{}
	file.Name = name + ".raw"
	file.ViewName = &file.Name
	file.DomainId = c.Session.DomainId
	file.Uuid = offer.Uuid
	file.UploadedBy = &model.Lookup{Id: int(c.Session.UserId)}
	file.Channel = model.NewString(model.UploadFileChannelScreenShare)

	file.SetEncrypted(false)
	file.GenerateThumbnail = false

	log := c.Log.With(wlog.String("name", file.Name))

	answer, err = c.App.UploadP2PVideo(offer.SessionDescription.SDP, file, offer.ICEServers)
	if err != nil {
		log.Error(err.Error())
	}

	w.WriteHeader(http.StatusOK)
	w.Write(encodeSdp(answer))

}

func encodeSdp(obj *webrtc.SessionDescription) []byte {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return b
}

func decodeSdp(in []byte, obj *RequestStoreScreen) error {
	return json.Unmarshal(in, obj)
}
