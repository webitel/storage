package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
	"github.com/webitel/webitel-go-kit/pkg/hostaddr"
	"github.com/webitel/wlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GrpcServer struct {
	srv *grpc.Server
	lis net.Listener
}

func (grpc *GrpcServer) AdvertiseAddr() (netip.AddrPort, error) {
	listen, err := netip.ParseAddrPort(grpc.lis.Addr().String())
	if err != nil {
		return netip.AddrPort{}, err
	}
	return hostaddr.Advertise(listen)
}

func unaryInterceptor(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	h, err := handler(ctx, req)

	if err != nil {
		wlog.Error(fmt.Sprintf("method %s duration %s, error: %v", info.FullMethod, time.Since(start), err.Error()))

		switch err.(type) {
		case model.AppError:
			e := err.(model.AppError)
			// ! TODO: e.Text() was here -->
			// func (er *AppError) Text() string {
			// 	if er.DetailedError != "" {
			// 		return er.DetailedError
			// 	}
			// 	return er.Message
			// }
			return h, wrapGrpcErr(e)
		default:
			return h, err
		}
	} else {
		wlog.Debug(fmt.Sprintf("method %s duration %s", info.FullMethod, time.Since(start)))
	}

	return h, err
}

func streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()

	err := handler(srv, ss)

	if err != nil {
		wlog.Error(fmt.Sprintf("method %s duration %s, error: %v", info.FullMethod, time.Since(start), err.Error()))

		switch err.(type) {
		case model.AppError:
			e := err.(model.AppError)
			return wrapGrpcErr(e)
		default:
			return err
		}
	} else {
		wlog.Debug(fmt.Sprintf("method %s duration %s", info.FullMethod, time.Since(start)))
	}

	return err
}

func wrapGrpcErr(err model.AppError) error {

	if model.IsFilePolicyError(err) { // WTEL-6931
		return status.Error(codes.FailedPrecondition, err.GetId())
	}

	var code codes.Code

	switch err.GetStatusCode() {
	case http.StatusBadRequest:
		code = codes.InvalidArgument
	case http.StatusAccepted:
		code = codes.ResourceExhausted
	case http.StatusUnauthorized:
		code = codes.Unauthenticated
	case http.StatusForbidden:
		code = codes.PermissionDenied
	case http.StatusNotFound:
		code = codes.NotFound
	default:

		code = codes.Internal

	}

	return status.Error(code, err.ToJson())
}

func NewGrpcServer(settings model.ServerSettings, log *wlog.Logger) (*GrpcServer, error) {
	host := strings.Trim(settings.Address, "[]")
	if host == "" {
		ip, err := hostaddr.PrivateIPv4()
		switch {
		case errors.Is(err, hostaddr.ErrNoPrivateIPv4):
			host = "127.0.0.1"
			log.Warn("no private IPv4 address found, grpc listens on loopback; set GRPC_ADDR to accept remote connections", wlog.String("address", host))
		case err != nil:
			return nil, fmt.Errorf("resolve default grpc address: %w; set GRPC_ADDR explicitly", err)
		default:
			host = ip.String()
		}
	}

	lis, err := net.Listen(settings.Network, net.JoinHostPort(host, strconv.Itoa(settings.Port)))
	if err != nil {
		return nil, err
	}
	return &GrpcServer{
		lis: lis,
		srv: grpc.NewServer(
			grpc.UnaryInterceptor(unaryInterceptor),
			grpc.StreamInterceptor(streamInterceptor),
		),
	}, nil
}

func (s *GrpcServer) Server() *grpc.Server {
	return s.srv
}

func (a *App) StartGrpcServer() error {
	go func() {
		defer wlog.Debug(fmt.Sprintf("[grpc] close server listening"))
		wlog.Debug(fmt.Sprintf("[grpc] server listening %s", a.GrpcServer.lis.Addr().String()))
		err := a.GrpcServer.srv.Serve(a.GrpcServer.lis)
		if err != nil {
			//FIXME
			panic(err.Error())
		}
	}()

	return nil
}

func tokenFromGrpcContext(ctx context.Context) (string, model.AppError) {
	if info, ok := metadata.FromIncomingContext(ctx); !ok {
		return "", model.NewInternalError("app.grpc.get_context", "Not found")
	} else {
		token := info.Get(model.HEADER_TOKEN)
		if len(token) < 1 {
			return "", model.NewInternalError("api.context.session_expired.app_error", "token not found")
		}
		return token[0], nil
	}
}

func (a *App) GetSessionFromCtx(ctx context.Context) (*auth_manager.Session, model.AppError) {
	var session *auth_manager.Session
	var err model.AppError
	var token string

	token, err = tokenFromGrpcContext(ctx)
	if err != nil {
		return nil, err
	}

	session, err = a.GetSession(token)
	if err != nil {
		return nil, err
	}

	if session.IsExpired() {
		return nil, model.NewInternalError("api.context.session_expired.app_error", "token="+token)
	}

	return session, nil
}
