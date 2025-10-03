package limiter

import (
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	LoadConcurrent int           `yaml:"load_concurrent" env-default:"10"`
	ReadConcurrent int           `yaml:"read_concurrent" env-default:"100"`
	TTL            time.Duration `yaml:"client_idle_ttl" env-default:"10m"`
}

type ClientLimiter struct {
	uploadSem chan struct{}
	listSem   chan struct{}

	uploadCount int64
	listCount   int64
}

func newClientLimiter(config *Config) *ClientLimiter {
	return &ClientLimiter{
		uploadSem: make(chan struct{}, config.LoadConcurrent),
		listSem:   make(chan struct{}, config.ReadConcurrent),
	}
}

func (c *ClientLimiter) AcquireUpload() bool {
	select {
	case c.uploadSem <- struct{}{}:
		atomic.AddInt64(&c.uploadCount, 1)
		return true

	default:
		return false
	}
}

func (c *ClientLimiter) ReleaseUpload() {
	select {
	case <-c.uploadSem:
		atomic.AddInt64(&c.uploadCount, -1)
	default:
	}
}

func (c *ClientLimiter) AcquireList() bool {
	select {
	case c.listSem <- struct{}{}:
		atomic.AddInt64(&c.listCount, 1)
		return true

	default:
		return false
	}
}

func (c *ClientLimiter) ReleaseList() {
	select {
	case <-c.listSem:
		atomic.AddInt64(&c.listCount, -1)
	default:
	}
}

type clientEntry struct {
	limiter  *ClientLimiter
	lastSeen time.Time
}

type Registry struct {
	mu      sync.Mutex
	clients map[string]*clientEntry
	//ttl     time.Duration
	config *Config
	stop   chan struct{}

	OnNewClient func(clientID string)
	OnPurge     func(clientID string)
}

func NewRegistry(config *Config) *Registry {
	r := &Registry{
		clients: make(map[string]*clientEntry),
		config:  config,
		stop:    make(chan struct{}),
	}

	go r.janitor()
	return r
}

func (r *Registry) Close() {
	close(r.stop)
}

func (r *Registry) janitor() {
	ticker := time.NewTicker(r.config.TTL / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			r.mu.Lock()

			for id, ent := range r.clients {
				if now.Sub(ent.lastSeen) > r.config.TTL {
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

func (r *Registry) Get(clientID string) *ClientLimiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ent, ok := r.clients[clientID]; ok {
		ent.lastSeen = time.Now()
		return ent.limiter
	}

	l := newClientLimiter(r.config)
	r.clients[clientID] = &clientEntry{
		limiter:  l,
		lastSeen: time.Now(),
	}
	if r.OnNewClient != nil {
		go r.OnNewClient(clientID)
	}
	return l
}
