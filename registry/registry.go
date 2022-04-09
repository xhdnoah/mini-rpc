package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// MiniRegistry is a simple register center, provide following functions.
// add a server and receive heartbeat to keep it alive.
// returns all alive servers and delete dead servers sync simultaneously.
type MiniRegistry struct {
	timeout time.Duration
	mu      sync.Mutex // protect following
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr  string
	start time.Time // 服务启动时间
}

const (
	defaultPath    = "/_minirpc_/registry"
	defaultTimeout = time.Minute * 5
)

// New create a registry instance with timeout setting
func New(timeout time.Duration) *MiniRegistry {
	return &MiniRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultMiniRegistry = New(defaultTimeout)

// 添加服务实例，如果已存在，则更新 start
func (r *MiniRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		s.start = time.Now() // if exists, update start time to keep alive
	}
}

// 返回可用服务列表，如果存在已超时服务，则删除
func (r *MiniRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) { // 未过期
			alive = append(alive, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

// Runs at /_minirpc_/registry 请求信息承载在 HTTP Header 中
func (r *MiniRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		// keep it simple, server is in req.Header
		w.Header().Set("X-Minirpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		// keep it simple, server is in req.Header
		addr := req.Header.Get("X-Minirpc-Addr")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// 添加服务实例
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP registers an HTTP handler for MiniRegistry messages on registryPath
func (r *MiniRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultMiniRegistry.HandleHTTP(defaultPath)
}

// Heartbeat send a heartbeat message every once in a while
// it's a helper function for a server to register or send heartbeat
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		// make sure there is enough time to send heart beat
		// before it's removed from registry
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Minirpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
