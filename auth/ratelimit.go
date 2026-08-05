package auth

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"time"

	"nouveauprintemps.org/atmail/utils"
)

type Limited struct {
	time   time.Time
	cancel chan struct{}

	grace uint8

	acc time.Duration
}

type RateLimiter struct {
	mu      sync.RWMutex
	limited map[netip.Addr]*Limited
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{limited: make(map[netip.Addr]*Limited)}
}

func (r *RateLimiter) IsLimited(ip net.IP) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	addr := netip.MustParseAddr(ip.String())
	v, ok := r.limited[addr]
	if !ok {
		return false
	}
	return v.time.After(time.Now())
}

func (r *RateLimiter) Limit(ctx context.Context, ip net.IP) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	l := utils.Logger(ctx)
	addr := netip.MustParseAddr(ip.String())
	v, ok := r.limited[addr]
	if !ok {
		r.limited[addr] = &Limited{cancel: make(chan struct{})}
		return false
	}
	if v.grace < 3 {
		v.grace++
		return false
	}
	close(v.cancel)
	prev := v.acc
	v.acc = max(v.acc, 2)
	v.acc = min(4*v.acc, 24*60*60)
	if v.acc != prev {
		l.Debug("rate limiting", "ip", ip, "for", v.acc*time.Second)
	}
	v.time = time.Now().Add(v.acc)
	v.cancel = make(chan struct{})
	go func(addr netip.Addr, acc time.Duration) {
		select {
		case <-time.Tick(4 * acc * time.Second):
			r.mu.Lock()
			defer r.mu.Unlock()
			delete(r.limited, addr)
			l.Debug("full cleaned", "ip", addr)
		case <-v.cancel:
		}
	}(addr, v.acc)
	return true
}

type LimiterListener struct {
	net.Listener
	Limiter *RateLimiter
}

func (l *LimiterListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		ip := conn.RemoteAddr().(*net.TCPAddr).IP
		if !l.Limiter.IsLimited(ip) {
			return conn, nil
		}
		conn.Close()
	}
}
