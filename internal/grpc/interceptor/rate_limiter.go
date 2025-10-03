package interceptor

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"fileservice/internal/limiter"
)

type ConcurrencyInterceptor struct {
	registry  *limiter.Registry
	OnAcquire func(method, clientID, kind string)
	OnRelease func(method, clientID, kind string)
	OnReject  func(method, clientID, kind string)
}

func New(registry *limiter.Registry) *ConcurrencyInterceptor {
	return &ConcurrencyInterceptor{
		registry: registry,
	}

}

func methodKind(fullMethod string) string {
	switch fullMethod {
	case "/file_service.FileService/ListFiles":
		return "list"

	case "/file_service.FileService/UploadFile",
		"/file_service.FileService/GetFile":

		return "load"
	default:
		return "load"
	}
}

func (ci *ConcurrencyInterceptor) getId(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if ok && p.Addr != nil {
		host, _, err := net.SplitHostPort(p.Addr.String())
		if err == nil && host != "" {
			return host
		}

		return p.Addr.String()
	}

	return "unknown"
}

func (ci *ConcurrencyInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		clientID := ci.getId(ctx)
		lim := ci.registry.Get(clientID)

		kind := methodKind(info.FullMethod)
		switch kind {
		case "list":
			if !lim.AcquireList() {
				if ci.OnReject != nil {
					go ci.OnReject(info.FullMethod, clientID, kind)
				}

				return nil, status.Error(codes.ResourceExhausted, "too many concurrent ListFiles requests")
			}

			if ci.OnAcquire != nil {
				go ci.OnAcquire(info.FullMethod, clientID, kind)
			}

			defer func() {
				lim.ReleaseList()
				if ci.OnRelease != nil {
					go ci.OnRelease(info.FullMethod, clientID, kind)
				}
			}()

		case "load":
			if !lim.AcquireUpload() {
				if ci.OnReject != nil {
					go ci.OnReject(info.FullMethod, clientID, kind)
				}

				return nil, status.Error(codes.ResourceExhausted, "too many concurrent Upload/Download requests")
			}

			if ci.OnAcquire != nil {
				go ci.OnAcquire(info.FullMethod, clientID, kind)
			}

			defer func() {
				lim.ReleaseUpload()
				if ci.OnRelease != nil {
					go ci.OnRelease(info.FullMethod, clientID, kind)
				}
			}()
		}

		return handler(ctx, req)
	}
}

func (ci *ConcurrencyInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		clientID := ci.getId(ss.Context())
		lim := ci.registry.Get(clientID)

		kind := methodKind(info.FullMethod)
		switch kind {
		case "list":
			if !lim.AcquireList() {
				if ci.OnReject != nil {
					go ci.OnReject(info.FullMethod, clientID, kind)
				}

				return status.Error(codes.ResourceExhausted, "too many concurrent ListFiles requests")
			}

			if ci.OnAcquire != nil {
				go ci.OnAcquire(info.FullMethod, clientID, kind)
			}

			defer func() {
				lim.ReleaseList()
				if ci.OnRelease != nil {
					go ci.OnRelease(info.FullMethod, clientID, kind)
				}
			}()

		case "load":
			if !lim.AcquireUpload() {
				if ci.OnReject != nil {
					go ci.OnReject(info.FullMethod, clientID, kind)
				}

				return status.Error(codes.ResourceExhausted, "too many concurrent Upload/Download requests")
			}

			if ci.OnAcquire != nil {
				go ci.OnAcquire(info.FullMethod, clientID, kind)
			}

			defer func() {
				lim.ReleaseUpload()
				if ci.OnRelease != nil {
					go ci.OnRelease(info.FullMethod, clientID, kind)
				}
			}()
		}

		return handler(srv, ss)
	}
}
