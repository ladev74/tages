//package limiter
//
//import (
//	"context"
//	"sync"
//
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/codes"
//	"google.golang.org/grpc/peer"
//	"google.golang.org/grpc/status"
//)
//
//type Config struct {
//	LoadConcurrent int `yaml:"load_concurrent" env-default:"10"`
//	ReadConcurrent int `yaml:"read_concurrent" env-default:"100"`
//}
//
//type Limiter struct {
//	loadConcurrent chan struct{}
//	readConcurrent chan struct{}
//}
//
//func New(config Config) *Limiter {
//	return &Limiter{
//		loadConcurrent: make(chan struct{}, config.LoadConcurrent),
//		readConcurrent: make(chan struct{}, config.ReadConcurrent),
//	}
//}
//
//func (l *Limiter) AcquireLoad() bool {
//	select {
//	case l.loadConcurrent <- struct{}{}:
//		return true
//	default:
//		return false
//	}
//}
//
//func (l *Limiter) ReleaseLoad() {
//	<-l.loadConcurrent
//}
//
//func (l *Limiter) AcquireRead() bool {
//	select {
//	case l.readConcurrent <- struct{}{}:
//		return true
//	default:
//		return false
//	}
//}
//
//func (l *Limiter) ReleaseReads() {
//	<-l.readConcurrent
//}
//
//type Registry struct {
//	mu      *sync.Mutex
//	clients map[string]*Limiter
//}
//
//func NewRegistry() *Registry {
//	return &Registry{
//		clients: make(map[string]*Limiter),
//	}
//}
//
//func (r *Registry) Get(clientID string, config Config) *Limiter {
//	r.mu.Lock()
//	defer r.mu.Unlock()
//
//	cl, ok := r.clients[clientID]
//	if !ok {
//		cl = New(config)
//		r.clients[clientID] = cl
//	}
//	return cl
//}
//
//type ConcurrencyInterceptor struct {
//	registry *Registry
//}
//
//func NewConcurrencyInterceptor(r *Registry) *ConcurrencyInterceptor {
//	return &ConcurrencyInterceptor{registry: r}
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
//			if !cl.AcquireRead() {
//				return nil, status.Error(codes.ResourceExhausted, "too many concurrent ListFiles requests")
//			}
//
//			release = cl.ReleaseReads
//
//		default:
//			if !cl.AcquireLoad() {
//				return nil, status.Error(codes.ResourceExhausted, "too many concurrent Upload/Download requests")
//			}
//
//			release = cl.ReleaseLoad
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
//			if !cl.AcquireRead() {
//				return status.Error(codes.ResourceExhausted, "too many concurrent ListFiles requests")
//			}
//
//			release = cl.ReleaseReads
//		default:
//
//			if !cl.AcquireLoad() {
//				return status.Error(codes.ResourceExhausted, "too many concurrent Upload/Download requests")
//			}
//
//			release = cl.ReleaseLoad
//		}
//
//		defer release()
//		return handler(srv, ss)
//	}
//}
//
//func (ci *ConcurrencyInterceptor) getClientID(ctx context.Context) string {
//	if p, ok := peer.FromContext(ctx); ok {
//		return p.Addr.String()
//	}
//
//	return "unknown"
//}

package limiter

import (
	"context"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Config struct {
	LoadConcurrent int `yaml:"load_concurrent" env-default:"10"`
	ReadConcurrent int `yaml:"read_concurrent" env-default:"100"`

	ClientIdleTTL time.Duration

	ClientIDFromMetadataKey string
}

type clientLimiter struct {
	uploadSem chan struct{}
	listSem   chan struct{}

	uploadCount int64
	listCount   int64
}

func newClientLimiter(config Config) *clientLimiter {
	return &clientLimiter{
		uploadSem: make(chan struct{}, config.LoadConcurrent),
		listSem:   make(chan struct{}, config.ReadConcurrent),
	}
}

func (c *clientLimiter) acquireUpload() bool {
	select {
	case c.uploadSem <- struct{}{}:
		atomic.AddInt64(&c.uploadCount, 1)
		return true
	default:
		return false
	}
}

func (c *clientLimiter) releaseUpload() {
	select {
	case <-c.uploadSem:
		atomic.AddInt64(&c.uploadCount, -1)
	default:
	}
}

func (c *clientLimiter) acquireList() bool {
	select {
	case c.listSem <- struct{}{}:
		atomic.AddInt64(&c.listCount, 1)
		return true
	default:
		return false
	}
}

func (c *clientLimiter) releaseList() {
	select {
	case <-c.listSem:
		atomic.AddInt64(&c.listCount, -1)
	default:
	}
}

type clientEntry struct {
	limiter  *clientLimiter
	lastSeen time.Time
}

type Registry struct {
	mu      sync.Mutex
	clients map[string]*clientEntry
	ttl     time.Duration
	stop    chan struct{}

	OnNewClient func(clientID string)
	OnPurge     func(clientID string)
}

func NewRegistry(idleTTL time.Duration) *Registry {
	r := &Registry{
		clients: make(map[string]*clientEntry),
		ttl:     idleTTL,
		stop:    make(chan struct{}),
	}
	go r.janitor()
	return r
}

func (r *Registry) Close() {
	close(r.stop)
}

func (r *Registry) janitor() {
	ticker := time.NewTicker(r.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			r.mu.Lock()
			for id, ent := range r.clients {
				if now.Sub(ent.lastSeen) > r.ttl {
					delete(r.clients, id)
					if r.OnPurge != nil {
						go r.OnPurge(id)
					}
				}
			}
			r.mu.Unlock()
		case <-r.stop:
			return
		}
	}
}

func (r *Registry) Get(clientID string, cfg Config) *clientLimiter {
	// Fast read lock
	r.mu.Lock()
	ent, ok := r.clients[clientID]
	r.mu.Unlock()
	if ok {
		r.mu.Lock()
		ent.lastSeen = time.Now()
		r.mu.Unlock()

		return ent.limiter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if ent, ok = r.clients[clientID]; ok {
		ent.lastSeen = time.Now()

		return ent.limiter
	}

	l := newClientLimiter(cfg)
	r.clients[clientID] = &clientEntry{
		limiter:  l,
		lastSeen: time.Now(),
	}

	if r.OnNewClient != nil {
		go r.OnNewClient(clientID)
	}

	return l
}

type ConcurrencyInterceptor struct {
	registry *Registry

	OnAcquire func(method, clientID, kind string)
	OnRelease func(method, clientID, kind string)
	OnReject  func(method, clientID, kind string)

	clientIDFunc func(ctx context.Context, mdKey string) string
}

func NewConcurrencyInterceptor(registry *Registry) *ConcurrencyInterceptor {
	ci := &ConcurrencyInterceptor{
		registry: registry,
	}

	ci.clientIDFunc = defaultClientIDFunc

	return ci
}

func defaultClientIDFunc(ctx context.Context, mdKey string) string {
	if mdKey != "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(mdKey); len(vals) > 0 && strings.TrimSpace(vals[0]) != "" {
				return strings.TrimSpace(vals[0])
			}
		}
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 && strings.TrimSpace(xff[0]) != "" {
			parts := strings.Split(xff[0], ",")

			return strings.TrimSpace(parts[0])
		}
	}

	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		host, _, err := net.SplitHostPort(p.Addr.String())
		if err == nil && host != "" {
			return host
		}

		return p.Addr.String()
	}

	return "unknown"
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

func (ci *ConcurrencyInterceptor) Unary(cfg Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		clientID := ci.clientIDFunc(ctx, cfg.ClientIDFromMetadataKey)
		lim := ci.registry.Get(clientID, cfg)

		kind := methodKind(info.FullMethod)
		switch kind {
		case "list":
			if !lim.acquireList() {
				if ci.OnReject != nil {
					go ci.OnReject(info.FullMethod, clientID, kind)
				}

				return nil, status.Error(codes.ResourceExhausted, "too many concurrent ListFiles requests")
			}

			if ci.OnAcquire != nil {
				go ci.OnAcquire(info.FullMethod, clientID, kind)
			}

			defer func() {
				lim.releaseList()
				if ci.OnRelease != nil {
					go ci.OnRelease(info.FullMethod, clientID, kind)
				}
			}()

		case "load":
			if !lim.acquireUpload() {
				if ci.OnReject != nil {
					go ci.OnReject(info.FullMethod, clientID, kind)
				}

				return nil, status.Error(codes.ResourceExhausted, "too many concurrent Upload/Download requests")
			}

			if ci.OnAcquire != nil {
				go ci.OnAcquire(info.FullMethod, clientID, kind)
			}

			defer func() {
				lim.releaseUpload()
				if ci.OnRelease != nil {
					go ci.OnRelease(info.FullMethod, clientID, kind)
				}
			}()
		}

		return handler(ctx, req)
	}
}

func (ci *ConcurrencyInterceptor) Stream(cfg Config) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		clientID := ci.clientIDFunc(ss.Context(), cfg.ClientIDFromMetadataKey)
		lim := ci.registry.Get(clientID, cfg)

		kind := methodKind(info.FullMethod)
		switch kind {
		case "list":
			if !lim.acquireList() {
				if ci.OnReject != nil {
					go ci.OnReject(info.FullMethod, clientID, kind)
				}

				return status.Error(codes.ResourceExhausted, "too many concurrent ListFiles requests")
			}

			if ci.OnAcquire != nil {
				go ci.OnAcquire(info.FullMethod, clientID, kind)
			}

			defer func() {
				lim.releaseList()
				if ci.OnRelease != nil {
					go ci.OnRelease(info.FullMethod, clientID, kind)
				}
			}()

		case "load":
			if !lim.acquireUpload() {
				if ci.OnReject != nil {
					go ci.OnReject(info.FullMethod, clientID, kind)
				}

				return status.Error(codes.ResourceExhausted, "too many concurrent Upload/Download requests")
			}

			if ci.OnAcquire != nil {
				go ci.OnAcquire(info.FullMethod, clientID, kind)
			}

			defer func() {
				lim.releaseUpload()
				if ci.OnRelease != nil {
					go ci.OnRelease(info.FullMethod, clientID, kind)
				}
			}()
		}

		return handler(srv, ss)
	}
}
