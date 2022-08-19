package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/engine/utils"
	"github.com/webitel/storage/model"
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

func (grpc *GrpcServer) GetPublicInterface() (string, int) {
	h, p, _ := net.SplitHostPort(grpc.lis.Addr().String())
	if h == "::" {
		h = utils.GetPublicAddr()
	}
	port, _ := strconv.Atoi(p)
	return h, port
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
		case *model.AppError:
			e := err.(*model.AppError)
			return h, status.Error(httpCodeToGrpc(e.StatusCode), e.Text())
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
		case *model.AppError:
			e := err.(*model.AppError)
			return status.Error(httpCodeToGrpc(e.StatusCode), e.Text())
		default:
			return err
		}
	} else {
		wlog.Debug(fmt.Sprintf("method %s duration %s", info.FullMethod, time.Since(start)))
	}

	return err
}

func httpCodeToGrpc(c int) codes.Code {
	switch c {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusAccepted:
		return codes.ResourceExhausted
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	default:
		return codes.Internal

	}
}

func NewGrpcServer(settings model.ServerSettings) *GrpcServer {
	address := fmt.Sprintf("%s:%d", settings.Address, settings.Port)
	lis, err := net.Listen(settings.Network, address)
	if err != nil {
		panic(err.Error())
	}
	return &GrpcServer{
		lis: lis,
		srv: grpc.NewServer(
			grpc.UnaryInterceptor(unaryInterceptor),
			grpc.StreamInterceptor(streamInterceptor),
		),
	}
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

func (a *App) GetSessionFromCtx(ctx context.Context) (*auth_manager.Session, *model.AppError) {
	var session *auth_manager.Session
	var err *model.AppError
	var token []string

	if info, ok := metadata.FromIncomingContext(ctx); !ok {
		return nil, model.NewAppError("GetSessionFromCtx", "app.grpc.get_context", nil, "Not found", http.StatusInternalServerError)
	} else {
		token = info.Get(model.HEADER_TOKEN)
	}

	if len(token) < 1 {
		return nil, model.NewAppError("GetSessionFromCtx", "api.context.session_expired.app_error", nil, "token not found", http.StatusUnauthorized)
	}

	session, err = a.GetSession(token[0])
	if err != nil {
		return nil, err
	}

	if session.IsExpired() {
		return nil, model.NewAppError("GetSessionFromCtx", "api.context.session_expired.app_error", nil, "token="+token[0], http.StatusUnauthorized)
	}

	return session, nil
}
