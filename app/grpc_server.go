package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
	"github.com/webitel/webitel-go-kit/infra/otel/correlation"
	"github.com/webitel/webitel-go-kit/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const grpcLogComponent logging.Component = "storage.grpc"

type GrpcServer struct {
	srv    *grpc.Server
	lis    net.Listener
	logger *logging.Logger
}

type sessionContextKey struct{}

type sessionResult struct {
	session *auth_manager.Session
	err     model.AppError
}

type authServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authServerStream) Context() context.Context {
	return s.ctx
}

func (s *GrpcServer) GetPublicInterface() (string, int) {
	h, p, _ := net.SplitHostPort(s.lis.Addr().String())
	if h == "::" {
		h = model.GetPublicAddr()
	}
	port, _ := strconv.Atoi(p)
	return h, port
}

func (s *GrpcServer) Server() *grpc.Server {
	return s.srv
}

func (a *App) authContext(ctx context.Context) context.Context {
	token, err := tokenFromGrpcContext(ctx)
	if err != nil {
		return ctx
	}

	session, err := a.GetSession(ctx, token)
	ctx = context.WithValue(ctx, sessionContextKey{}, sessionResult{session: session, err: err})
	if err != nil {
		return ctx
	}

	ctx = correlation.WithIdentity(ctx, session.DomainId, session.UserId)
	return ctx
}

func (a *App) unaryInterceptor(logger *logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		ctx = a.authContext(ctx)

		h, err := handler(ctx, req)

		if err != nil {
			logger.ErrorCtx(ctx, "grpc call failed",
				logging.Str("method", info.FullMethod),
				logging.Dur("duration", time.Since(start)),
				logging.Err(err),
			)

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
			logger.DebugCtx(ctx, "grpc call",
				logging.Str("method", info.FullMethod),
				logging.Dur("duration", time.Since(start)),
			)
		}

		return h, err
	}
}

func (a *App) streamInterceptor(logger *logging.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := a.authContext(ss.Context())

		err := handler(srv, &authServerStream{ServerStream: ss, ctx: ctx})

		if err != nil {
			logger.ErrorCtx(ctx, "grpc stream failed",
				logging.Str("method", info.FullMethod),
				logging.Dur("duration", time.Since(start)),
				logging.Err(err),
			)

			switch err.(type) {
			case model.AppError:
				e := err.(model.AppError)
				return wrapGrpcErr(e)
			default:
				return err
			}
		} else {
			logger.DebugCtx(ctx, "grpc stream",
				logging.Str("method", info.FullMethod),
				logging.Dur("duration", time.Since(start)),
			)
		}

		return err
	}
}

func (a *App) newGrpcServer(settings model.ServerSettings) *GrpcServer {
	address := fmt.Sprintf("%s:%d", settings.Address, settings.Port)
	lis, err := net.Listen(settings.Network, address)
	if err != nil {
		panic(err.Error())
	}
	logger := a.logger.WithComponent(grpcLogComponent)
	return &GrpcServer{
		lis:    lis,
		logger: logger,
		srv: grpc.NewServer(
			grpc.UnaryInterceptor(a.unaryInterceptor(logger)),
			grpc.StreamInterceptor(a.streamInterceptor(logger)),
		),
	}
}

func (a *App) StartGrpcServer() error {
	go func() {
		ctx := context.Background()
		addrField := logging.Str("address", a.GrpcServer.lis.Addr().String())
		defer a.GrpcServer.logger.DebugCtx(ctx, "grpc server closed", addrField)
		a.GrpcServer.logger.DebugCtx(ctx, "grpc server listening", addrField)
		err := a.GrpcServer.srv.Serve(a.GrpcServer.lis)
		if err != nil {
			//FIXME
			panic(err.Error())
		}
	}()

	return nil
}

func (a *App) GetSessionFromCtx(ctx context.Context) (*auth_manager.Session, model.AppError) {
	var session *auth_manager.Session
	var err model.AppError
	var token string

	token, err = tokenFromGrpcContext(ctx)
	if err != nil {
		return nil, err
	}

	if r, ok := ctx.Value(sessionContextKey{}).(sessionResult); ok {
		session, err = r.session, r.err
	} else {
		session, err = a.GetSession(ctx, token)
	}
	if err != nil {
		return nil, err
	}

	if session.IsExpired() {
		return nil, model.NewInternalError("api.context.session_expired.app_error", "token="+token)
	}

	return session, nil
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
