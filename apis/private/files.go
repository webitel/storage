package private

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

func (api *API) InitFile() {
	api.Routes.Files.Handle("", api.ApiHandler(putRecordCallFile)).Methods("PUT")
	api.Routes.Files.Handle("/ai/{id}", api.ApiHandler(putAiSpeech)).Methods("PUT")
	api.Routes.Files.Handle("/ai/{id}", api.ApiHandler(getAiSpeech)).Methods("GET")
	api.Routes.Files.Handle("/ai/{id}/metadata", api.ApiHandler(getAiMetadata)).Methods("GET")
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
	fileRequest.Channel = model.NewString(model.UploadFileChannelCall)

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

func getAiMetadata(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
		return
	}

	ai, err := RecoverySafeAi(c.Params.Id)
	if err != nil {
		wlog.Error(err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	m := make(map[string]string)
	tmp, _ := base64.StdEncoding.DecodeString(ai.res.Header.Get("Human"))
	m["ai_human"] = string(tmp)

	tmp, _ = base64.StdEncoding.DecodeString(ai.res.Header.Get("Bot_answer"))
	m["ai_answer"] = string(tmp)

	tmp, _ = base64.StdEncoding.DecodeString(ai.res.Header.Get("Context"))
	m["ai_context"] = string(tmp)
	ai.Wait(time.Second * 2)

	res, _ := json.Marshal(m)

	w.WriteHeader(http.StatusOK)
	w.Write(res)

}

func getAiSpeech(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
		return
	}

	ai, err := RecoverySafeAi(c.Params.Id)
	if err != nil {
		wlog.Error(err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer func() {
		ai.Close()
	}()

	w.WriteHeader(http.StatusOK)
	io.Copy(w, ai.res.Body)

}

func putAiSpeech(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
		return
	}
	id := c.Params.Id
	wlog.Debug(fmt.Sprintf("start record %s", id))
	defer wlog.Debug(fmt.Sprintf("stop record %s", id))

	q := r.URL.Query()
	addresses, _ := base64.URLEncoding.DecodeString(q.Get("addresses"))
	context, _ := base64.URLEncoding.DecodeString(q.Get("context"))
	chatHistory, _ := base64.URLEncoding.DecodeString(q.Get("chat_history"))
	url := q.Get("url")

	body, writer := io.Pipe()

	req, err := http.NewRequest(http.MethodPost, url, body)

	if err != nil {
		wlog.Error(err.Error())
		return
	}

	mwriter := multipart.NewWriter(writer)
	req.Header.Add("Content-Type", mwriter.FormDataContentType())

	errchan := make(chan error)

	go func() {
		defer close(errchan)
		defer writer.Close()
		defer mwriter.Close()

		a, err := mwriter.CreateFormField("addresses")
		a.Write([]byte(base64.StdEncoding.EncodeToString(addresses)))
		a, err = mwriter.CreateFormField("context")
		a.Write([]byte(base64.StdEncoding.EncodeToString(context)))
		a, err = mwriter.CreateFormField("chat_history")
		a.Write([]byte(base64.StdEncoding.EncodeToString(chatHistory)))

		w, err := mwriter.CreateFormFile("file", id+".mp3")
		if err != nil {
			errchan <- err
			return
		}

		if written, err := io.Copy(w, r.Body); err != nil {
			errchan <- fmt.Errorf("error copying %s (%d bytes written): %v", id, written, err)
			return
		}

		if err := mwriter.Close(); err != nil {
			errchan <- err
			return
		}
	}()

	resp, err := http.DefaultClient.Do(req)
	merr := <-errchan
	body.Close()

	if err != nil || merr != nil {
		wlog.Error(fmt.Sprintf("http error: %v / %v", err, merr))
		return
	}

	if resp.StatusCode != http.StatusOK {
		v, _ := io.ReadAll(resp.Body)
		wlog.Error(string(v))
		resp.Body.Close()
	} else {
		StoreSafeAi(id, resp, time.Second*2)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"status\": \"+OK\"}"))
}
