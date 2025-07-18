package app

import (
	"context"
	"fmt"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/h264writer"
	"github.com/pion/webrtc/v4/pkg/media/ivfwriter"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

var webrtcAPI *webrtc.API

type SessionDescription = webrtc.SessionDescription
type ICEServer = webrtc.ICEServer

var debugRtp = os.Getenv("DEBUG_RTP") == "true"

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

	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}

	if debugRtp {
		log.Debug("sdp offer:\n" + sdpOffer)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var writer media.Writer

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) { //nolint: revive
		codec := track.Codec()
		log.Debug(fmt.Sprintf("got %s track, saving as %s", codec.MimeType, file.Name))

		rd, wd := io.Pipe()

		go func() {
			defer func() {
				log.Debug("closing pipe writer")
				wd.Close()
			}()

			// TODO
			//src, err := app.FilePolicyForUpload(file.DomainId, &file.BaseFile, rd)
			//if err != nil {
			//	// error
			//}

			if appErr := app.SyncUpload(rd, file); appErr != nil {
				log.Error(appErr.Error())
				cancel()
			} else {
				//appErr = app.Store.SyncFile().CreateJob(file.DomainId, file.Id, model.Transcoding, nil)
				//if appErr != nil {
				//	log.Error(appErr.Error())
				//}
				log.Debug("file upload finished successfully")
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
				select {
				case <-ctx.Done():
					log.Debug("context canceled, stopping rtp reader loop")
					return
				default:
				}

				rtpPacket, _, err = track.ReadRTP()
				if err != nil {
					if err != io.EOF {
						log.Error(fmt.Sprintf("unhandled error reading rtp packet: %s", err))
					}
					cancel()
					return
				}

				if debugRtp {
					//log.Debug(fmt.Sprintf("rtp ts=%d seq=%d", rtpPacket.Timestamp, rtpPacket.SequenceNumber))
				}

				if err = writer.WriteRTP(rtpPacket); err != nil {
					log.Error(fmt.Sprintf("failed to write rtp packet: %s", err))
					cancel()
					return
				}
			}
		}
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Debug(fmt.Sprintf("connection state has changed to %s", connectionState.String()))

		if connectionState == webrtc.ICEConnectionStateFailed || connectionState == webrtc.ICEConnectionStateClosed ||
			connectionState == webrtc.ICEConnectionStateDisconnected {

			log.Info(fmt.Sprintf("peer connection closed/failed (%s), starting cleanup", connectionState.String()))
			cancel()

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

	sd := peerConnection.LocalDescription()
	if debugRtp {
		log.Debug("sdp answer:\n" + sd.SDP)
	}

	return sd, nil
}

func init() {
	mediaEngine := &webrtc.MediaEngine{}

	s := webrtc.SettingEngine{}
	s.SetICETimeouts(5*time.Second, 30*time.Second, 5*time.Second)
	s.SetIPFilter(func(ip net.IP) bool {
		return ip.To4() != nil
	})

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

	registry := &interceptor.Registry{}

	// Register a intervalpli factory
	// This interceptor sends a PLI every 3 seconds. A PLI causes a video keyframe to be generated by the sender.
	// This makes our video seekable and more error resilent, but at a cost of lower picture quality and higher bitrates
	// A real world application should process incoming RTCP packets from viewers and forward them to senders
	intervalPliFactory, err := intervalpli.NewReceiverInterceptor()
	if err != nil {
		panic(err)
	}
	registry.Add(intervalPliFactory)

	// Use the default set of Interceptors
	if err = webrtc.RegisterDefaultInterceptors(mediaEngine, registry); err != nil {
		panic(err)
	}

	webrtcAPI = webrtc.NewAPI(
		webrtc.WithSettingEngine(s),
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithInterceptorRegistry(registry))
}
