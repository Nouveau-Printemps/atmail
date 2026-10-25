package auth

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"nouveauprintemps.org/atmail/utils"
)

type Limited struct {
	limited bool
	ip      netip.Addr
	cancel  chan struct{}
	acc     atomic.Uint32
}

func (l *Limited) increase(logger *slog.Logger, r *RateLimiter) {
	acc := l.acc.Add(1)
	if l.cancel != nil {
		close(l.cancel)
	}
	l.cancel = make(chan struct{})
	bon := rand.Uint32N(acc * 2)
	select {
	case <-l.cancel:
	case <-time.After(5*time.Minute + time.Duration(bon)*time.Minute):
		l.limited = false
		logger.Debug("disable ratelimiting")
		// save it for a certain time (at least 1h) to catch slow scanners
		select {
		case <-l.cancel:
		case <-time.After(1*time.Hour + time.Duration(bon)*time.Minute*30):
			r.mu.Lock()
			defer r.mu.Unlock()
			delete(r.limited, l.ip)
			logger.Debug("fully cleaned")
		}
	}
}

type RateLimiter struct {
	mu      sync.RWMutex
	limited map[netip.Addr]*Limited
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{limited: make(map[netip.Addr]*Limited)}
}

func (r *RateLimiter) IsLimited(ip net.IP) *Limited {
	r.mu.RLock()
	defer r.mu.RUnlock()
	addr := netip.MustParseAddr(ip.String())
	v, ok := r.limited[addr]
	if !ok || !v.limited {
		return nil
	}
	return v
}

func (r *RateLimiter) Accept(ctx context.Context, ip net.IP) {
	r.mu.Lock()
	defer r.mu.Unlock()
	addr := netip.MustParseAddr(ip.String())
	v, ok := r.limited[addr]
	if !ok {
		return
	}
	utils.Logger(ctx).Debug("accept connection")
	close(v.cancel)
	delete(r.limited, addr)
}

func (r *RateLimiter) Limit(ctx context.Context, ip net.IP) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	l := utils.Logger(ctx)
	addr := netip.MustParseAddr(ip.String())
	v, ok := r.limited[addr]
	if !ok {
		v = &Limited{ip: addr}
		r.limited[addr] = v
		l.Debug("monitoring sus connection")
	} else {
		l.Debug("rate limiting", "acc", v.acc.Load())
	}
	v.limited = ok
	go v.increase(l, r)
	return ok
}

type LimiterListener struct {
	net.Listener
	Limiter *RateLimiter
	Logger  *slog.Logger
}

func (l *LimiterListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		ip := conn.RemoteAddr().(*net.TCPAddr).IP
		v := l.Limiter.IsLimited(ip)
		if v == nil {
			return conn, nil
		}
		log := l.Logger.With("addr", conn.RemoteAddr())
		v.limited = true
		log.Debug("rate limiting", "acc", v.acc.Load())
		go v.increase(log, l.Limiter)
		conn.Close()
	}
}
