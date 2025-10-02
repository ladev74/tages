package interceptor

//
//import (
//	"sync"
//
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/codes"
//	"google.golang.org/grpc/peer"
//	"google.golang.org/grpc/status"
//
//	"fileservice/internal/limiter"
//)
//
//type Registry struct {
//	mu      *sync.Mutex
//	clients map[string]*limiter.Limiter
//}
//
//func New() *Registry {
//	return &Registry{
//		clients: make(map[string]*limiter.Limiter),
//	}
//}
//
//func (r *Registry) Get(clientID string, config limiter.Config) *limiter.Limiter {
//	r.mu.Lock()
//	defer r.mu.Unlock()
//
//	lim, ok := r.clients[clientID]
//	if !ok {
//		lim = limiter.New(config)
//		r.clients[clientID] = lim
//	}
//
//	return lim
//}
//
//type ConcurrencyInterceptor struct {
//	registry *Registry
//}
//
//func NewConcurrencyInterceptor(r *Registry) *ConcurrencyInterceptor {
//	return &ConcurrencyInterceptor{
//		registry: r,
//	}
//}
//
//func (ci *ConcurrencyInterceptor) Unary(config Config) grpc.UnaryServerInterceptor {
//	return func(
//		ctx context.Context,
//		req interface{},
//		info *grpc.UnaryServerInfo,
//		handler grpc.UnaryHandler,
//	) (resp interface{}, err error) {
//		clientID := ci.getClientID(ctx)
//		cl := ci.registry.Get(clientID, config)
//
//		release := func() {}
//		switch {
//		case info.FullMethod == "/file_service.FileService/ListFiles":
//			if !cl.AcquireListFiles() {
//				return nil, status.Error(codes.ResourceExhausted, "too many concurrent ListFiles requests")
//			}
//			release = cl.ReleaseListFiles
//		default:
//			// все unary методы кроме ListFiles считаем Upload/Download
//			if !cl.AcquireUploadDownload() {
//				return nil, status.Error(codes.ResourceExhausted, "too many concurrent Upload/Download requests")
//			}
//			release = cl.ReleaseUploadDownload
//		}
//
//		defer release()
//		return handler(ctx, req)
//	}
//}
//
//func (ci *ConcurrencyInterceptor) Stream(config Config) grpc.StreamServerInterceptor {
//	return func(
//		srv interface{},
//		ss grpc.ServerStream,
//		info *grpc.StreamServerInfo,
//		handler grpc.StreamHandler,
//	) error {
//		clientID := ci.getClientID(ss.Context())
//		cl := ci.registry.Get(clientID, config)
//
//		release := func() {}
//		switch {
//		case info.FullMethod == "/file_service.FileService/ListFiles":
//			if !cl.AcquireListFiles() {
//				return status.Error(codes.ResourceExhausted, "too many concurrent ListFiles requests")
//			}
//			release = cl.ReleaseListFiles
//		default:
//			if !cl.AcquireUploadDownload() {
//				return status.Error(codes.ResourceExhausted, "too many concurrent Upload/Download requests")
//			}
//			release = cl.ReleaseUploadDownload
//		}
//
//		defer release()
//		return handler(srv, ss)
//	}
//}
//
//func (ci *ConcurrencyInterceptor) getClientID(ctx context.Context) string {
//	if p, ok := peer.FromContext(ctx); ok {
//		return p.Addr.String() // клиент определяется по IP:port
//	}
//	return "unknown"
//}
