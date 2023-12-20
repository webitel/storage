package private

import (
	"net/http"
	"strconv"

	"github.com/webitel/storage/model"
)

func (api *API) InitFile() {
	api.Routes.Files.Handle("", api.ApiHandler(putRecordCallFile)).Methods("PUT")
	api.Routes.Files.Handle("", api.ApiHandler(putRecordCallFile)).Methods("POST")
}

//	/sys/records?
//
// domain=10.10.10.144
// &id=65d252ab-3f9d-4293-b680-0728bb566acc
// &type=mp3
// &email=none
// &name=recordSession
// &email_sbj=none
// &email_msg=none
func putRecordCallFile(c *Context, w http.ResponseWriter, r *http.Request) {
	var fileRequest model.JobUploadFile
	var domainId int
	var err error

	if domainId, err = strconv.Atoi(r.URL.Query().Get("domain")); err != nil {
		c.SetInvalidUrlParam("domain")
		return
	}

	fileRequest.DomainId = int64(domainId)
	fileRequest.Uuid = r.URL.Query().Get("id")
	fileRequest.Name = r.URL.Query().Get("name")
	fileRequest.ViewName = &fileRequest.Name
	fileRequest.MimeType = r.Header.Get("Content-Type")

	if r.URL.Query().Get("email_msg") != "" && r.URL.Query().Get("email_msg") != "none" {
		fileRequest.EmailMsg = r.URL.Query().Get("email_msg")
		fileRequest.EmailSub = r.URL.Query().Get("email_sbj")
	}

	defer r.Body.Close()

	if err := c.App.AddUploadJobFile(r.Body, &fileRequest); err != nil {
		c.Err = err
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"status\": \"+OK\"}"))
}
