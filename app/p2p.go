package app

import (
	"context"
	"fmt"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/h264writer"
	"github.com/pion/webrtc/v4/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"io"
	"net"
	"os"
	"time"
)

var webrtcAPI *webrtc.API

var webRTCSessions = utils.NewLru(400)

type SessionDescription = webrtc.SessionDescription
type ICEServer = webrtc.ICEServer

type RtcUploadVideoSession struct {
	Id     string
	answer *SessionDescription
	offer  SessionDescription
	pc     *webrtc.PeerConnection
	log    *wlog.Logger
	file   *model.JobUploadFile
	cancel context.CancelFunc
	ctx    context.Context
	app    *App
	writer media.Writer
}

var debugRtp = os.Getenv("DEBUG_RTP") == "true"

func NewWebRtcUploadSession(app *App, pc *webrtc.PeerConnection, file *model.JobUploadFile) *RtcUploadVideoSession {
	session := &RtcUploadVideoSession{
		Id:   model.NewId(),
		file: file,
		pc:   pc,
		app:  app,
		log:  app.Log.With(wlog.String("name", file.Name)),
	}
	session.ctx, session.cancel = context.WithCancel(context.Background())
	pc.OnTrack(session.onTrack)
	pc.OnICEConnectionStateChange(session.onICEConnectionStateChange)

	return session
}

func (s *RtcUploadVideoSession) onTrack(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	codec := track.Codec()
	var err error
	var pkt rtp.Depacketizer

	s.log.Debug(fmt.Sprintf("got %s track, saving as %s", codec.MimeType, s.file.Name))

	rd, wd := io.Pipe()

	go func() {
		defer func() {
			s.log.Debug("closing pipe writer")
			wd.Close()
		}()

		// TODO
		//src, err := app.FilePolicyForUpload(file.DomainId, &file.BaseFile, rd)
		//if err != nil {
		//	// error
		//}

		if appErr := s.app.SyncUpload(rd, s.file); appErr != nil {
			s.log.Error(appErr.Error())
			s.cancel()
		} else {
			//appErr = app.Store.SyncFile().CreateJob(file.DomainId, file.Id, model.Transcoding, nil)
			//if appErr != nil {
			//	log.Error(appErr.Error())
			//}
			s.log.Debug("file upload finished successfully")
		}
	}()

	switch codec.MimeType {
	case webrtc.MimeTypeVP9:
		s.file.MimeType = codec.MimeType
		pkt = &codecs.VP9Packet{}
		s.writer, err = ivfwriter.NewWith(wd, ivfwriter.WithCodec(codec.MimeType))
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to open ivf file: %s", err))
			return
		}

	case webrtc.MimeTypeH264:
		s.file.MimeType = codec.MimeType
		pkt = &codecs.H264Packet{}
		s.writer = h264writer.NewWith(wd)
	}

	if s.writer != nil {

		var rtpPacket *rtp.Packet
		var sample *media.Sample
		var lsn uint16 = 0

		builder := samplebuilder.New(45, pkt, codec.ClockRate,
			samplebuilder.WithRTPHeaders(true),
			samplebuilder.WithPacketReleaseHandler(func(pkt *rtp.Packet) {
				//if debugRtp {
				//s.log.Debug(fmt.Sprintf("rtp ts=%d seq=%d", pkt.Timestamp, pkt.SequenceNumber))
				//}

				if lsn != 0 && pkt.SequenceNumber != lsn+1 {
					s.log.Error(fmt.Sprintf("lost packets packet seq=%d, last=%d, count=%d", pkt.SequenceNumber,
						lsn, pkt.SequenceNumber-(lsn+1)))
				}
				lsn = pkt.SequenceNumber
				if err = s.writer.WriteRTP(pkt); err != nil {
					s.log.Error(fmt.Sprintf("failed to write rtp packet: %s", err))
					s.cancel()
				}
			}),
		)

		for {
			select {
			case <-s.ctx.Done():
				s.log.Debug("context canceled, stopping rtp reader loop")
				return
			default:

				rtpPacket, _, err = track.ReadRTP()
				if err != nil {
					if err != io.EOF {
						s.log.Error(fmt.Sprintf("unhandled error reading rtp packet: %s", err))
					}
					s.close()
					return
				}

				builder.Push(rtpPacket)
				for sample = builder.Pop(); sample != nil; sample = builder.Pop() {
					//if _, err = wd.Write(sample.Data); err != nil {
					//	log.Error(fmt.Sprintf("failed to write rtp packet: %s", err))
					//	cancel()
					//	return
					//}
				}
			}
		}
	}

}

func (s *RtcUploadVideoSession) onICEConnectionStateChange(connectionState webrtc.ICEConnectionState) {
	s.log.Debug(fmt.Sprintf("connection state has changed to %s", connectionState.String()))

	switch connectionState {
	case webrtc.ICEConnectionStateFailed:
		s.close()
	default:

	}
}

func (s *RtcUploadVideoSession) close() {
	s.cancel()

	if s.writer != nil {
		if closeErr := s.writer.Close(); closeErr != nil {
			s.log.Error(fmt.Sprintf("closing writer: %s", closeErr.Error()))
		}
	}

	// Gracefully shutdown the peer connection
	if closeErr := s.pc.Close(); closeErr != nil {
		s.log.Error(fmt.Sprintf("closing peer connection: %s", closeErr.Error()))
	}
	webRTCSessions.Remove(s.Id)
}

func (s *RtcUploadVideoSession) negotiate(sdpOffer string) error {
	s.offer = webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdpOffer,
	}
	if debugRtp {
		s.log.Debug("sdp offer:\n" + s.offer.SDP)
	}

	// Set the remote SessionDescription
	err := s.pc.SetRemoteDescription(s.offer)
	if err != nil {
		return err
	}

	// Create answer
	var answer webrtc.SessionDescription
	answer, err = s.pc.CreateAnswer(nil)
	if err != nil {
		return err
	}

	gatherComplete := webrtc.GatheringCompletePromise(s.pc)
	err = s.pc.SetLocalDescription(answer)
	if err != nil {
		return err
	}

	<-gatherComplete

	s.answer = s.pc.LocalDescription()
	if debugRtp {
		s.log.Debug("sdp answer:\n" + s.answer.SDP)
	}

	return nil
}

func (s *RtcUploadVideoSession) AnswerSDP() string {
	if s.answer != nil {
		return s.answer.SDP
	}

	return ""
}

func (app *App) UploadP2PVideo(sdpOffer string, file *model.JobUploadFile, ice []ICEServer) (*RtcUploadVideoSession, error) {
	var peerConnection *webrtc.PeerConnection
	var err error

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

	session := NewWebRtcUploadSession(app, peerConnection, file)

	err = session.negotiate(sdpOffer)
	if err != nil {
		session.close()
		return nil, err
	}

	webRTCSessions.Add(session.Id, session)

	return session, nil
}

func (app *App) RenegotiateP2P(id string, sdpOffer string) (*RtcUploadVideoSession, error) {
	session, ok := webRTCSessions.Get(id)
	if !ok {
		return nil, fmt.Errorf("p2p session with id %s not found", id)
	}
	sess := session.(*RtcUploadVideoSession)

	// TODO singleflight
	err := sess.negotiate(sdpOffer)
	if err != nil {
		sess.close()
		return nil, err
	}

	return sess, nil
}

func (app *App) CloseP2P(id string) error {
	session, ok := webRTCSessions.Get(id)
	if !ok {
		return fmt.Errorf("p2p session with id %s not found", id)
	}

	// TODO singleflight
	session.(*RtcUploadVideoSession).close()
	return nil
}

func init() {
	mediaEngine := &webrtc.MediaEngine{}

	s := webrtc.SettingEngine{}
	s.SetICETimeouts(5*time.Second, 15*time.Second, 5*time.Second)
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
