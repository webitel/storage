package app

import (
	"fmt"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/h264writer"
	"github.com/pion/webrtc/v4/pkg/media/ivfwriter"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
	"io"
	"strings"
)

var webrtcAPI *webrtc.API

type SessionDescription = webrtc.SessionDescription
type ICEServer = webrtc.ICEServer

func (app *App) UploadP2PVideo(sdpOffer string, file *model.JobUploadFile, ice []ICEServer) (*SessionDescription, error) {
	var answer SessionDescription
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdpOffer,
	}
	var peerConnection *webrtc.PeerConnection
	var err error

	log := app.Log.With(wlog.String("name", file.Name))

	config := webrtc.Configuration{
		ICEServers: ice,
	}

	peerConnection, err = webrtcAPI.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// Allow us to receive 1 audio track, and 1 video track
	//if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
	//	panic(err)
	//}
	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}

	var writer media.Writer

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) { //nolint: revive
		codec := track.Codec()
		log.Debug(fmt.Sprintf("got %s track, saving as %s", codec.MimeType, file.Name))
		rd, wd := io.Pipe()
		go func() {
			// TODO
			//src, err := app.FilePolicyForUpload(file.DomainId, &file.BaseFile, rd)
			//if err != nil {
			//	// error
			//}
			if appErr := app.SyncUpload(rd, file); appErr != nil {
				log.Error(appErr.Error())
			} else {
				//appErr = app.Store.SyncFile().CreateJob(file.DomainId, file.Id, model.Transcoding, nil)
				//if appErr != nil {
				//	log.Error(appErr.Error())
				//}
			}
		}()

		if strings.EqualFold(codec.MimeType, webrtc.MimeTypeVP9) {
			file.MimeType = codec.MimeType
			writer, err = ivfwriter.NewWith(wd, ivfwriter.WithCodec(codec.MimeType))
			if err != nil {
				log.Error(fmt.Sprintf("failed to open ivf file: %s", err))
				return
			}
		} else if strings.EqualFold(codec.MimeType, webrtc.MimeTypeH264) {
			file.MimeType = codec.MimeType
			writer = h264writer.NewWith(wd)
		}

		if writer != nil {

			var rtpPacket *rtp.Packet

			for {
				rtpPacket, _, err = track.ReadRTP()
				if err != nil {
					if err != io.EOF {
						log.Error(err.Error())
					}
					return
				}
				if err = writer.WriteRTP(rtpPacket); err != nil {
					log.Error(err.Error())
					return
				}
			}
		}
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Debug(fmt.Sprintf("connection state has changed %s \n", connectionState.String()))

		if connectionState == webrtc.ICEConnectionStateFailed || connectionState == webrtc.ICEConnectionStateClosed ||
			connectionState == webrtc.ICEConnectionStateDisconnected {

			if writer != nil {
				if closeErr := writer.Close(); closeErr != nil {
					log.Error(fmt.Sprintf("closing writer: %s", closeErr.Error()))
				}
			}

			// Gracefully shutdown the peer connection
			if closeErr := peerConnection.Close(); closeErr != nil {
				log.Error(fmt.Sprintf("closing peer connection: %s", closeErr.Error()))
			}
		}
	})

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		return nil, err
	}

	// Create answer
	answer, err = peerConnection.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		return nil, err
	}

	<-gatherComplete

	return peerConnection.LocalDescription(), nil
}

func init() {
	mediaEngine := &webrtc.MediaEngine{}
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeVP9,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeH264,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 97,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	//
	//if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
	//	RTPCodecCapability: webrtc.RTPCodecCapability{
	//		MimeType:     webrtc.MimeTypeOpus,
	//		ClockRate:    48000,
	//		Channels:     0,
	//		SDPFmtpLine:  "",
	//		RTCPFeedback: nil,
	//	},
	//	PayloadType: 111,
	//}, webrtc.RTPCodecTypeAudio); err != nil {
	//	panic(err)
	//}
	/*
		interceptorRegistry := &interceptor.Registry{}

		// Register a intervalpli factory
		// This interceptor sends a PLI every 3 seconds. A PLI causes a video keyframe to be generated by the sender.
		// This makes our video seekable and more error resilent, but at a cost of lower picture quality and higher bitrates
		// A real world application should process incoming RTCP packets from viewers and forward them to senders
		intervalPliFactory, err := intervalpli.NewReceiverInterceptor()
		if err != nil {
			panic(err)
		}
		interceptorRegistry.Add(intervalPliFactory)

		// Use the default set of Interceptors
		if err = webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
			panic(err)
		}
	*/

	// Create the API object with the MediaEngine
	webrtcAPI = webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
}
