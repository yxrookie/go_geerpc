package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type GeeRegistry struct {
	timeout time.Duration
	mu sync.Mutex
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr string
	start time.Time
}

const (
	defaultPath = "/_geerpc_/registry"
	defaultTimeout = time.Minute * 5
)

func New(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultGeeRegister = New(defaultTimeout)

func (r *GeeRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s != nil {
		s.start = time.Now()
	} else {
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	}
	
}

func (r *GeeRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var aliveservers []string
	for addr, server := range r.servers {
		if r.timeout == 0 || server.start.Add(r.timeout).After(time.Now()) {
			aliveservers = append(aliveservers, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(aliveservers)
	return aliveservers
}

func (r *GeeRegistry) ServeHTTP (w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		addr := req.Header.Get("X-Geerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP registers an HTTP handler for GeeRegistry messages on registryPath
func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultGeeRegister.HandleHTTP(defaultPath)
}

func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<- t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to register", registry)
	httpClinet := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Geerpc-Server", addr)
	if _, err := httpClinet.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}

