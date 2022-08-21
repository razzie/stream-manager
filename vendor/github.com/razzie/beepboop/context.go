package beepboop

import (
	"context"
	"log"
	"time"

	"github.com/razzie/geoip-server/geoip"
	"golang.org/x/time/rate"
)

// Context ...
type Context struct {
	Context          context.Context
	middlewares      []Middleware
	DB               *DB
	Logger           *log.Logger
	GeoIPClient      geoip.Client
	Limiters         map[string]*RateLimiter
	Layout           Layout
	CookieExpiration time.Duration
}

func newContext(ctx context.Context, layout Layout, srv *Server) *Context {
	return &Context{
		Context:          ctx,
		middlewares:      srv.Middlewares,
		DB:               srv.DB,
		Logger:           srv.Logger,
		GeoIPClient:      srv.GeoIPClient,
		Limiters:         srv.Limiters,
		Layout:           layout,
		CookieExpiration: srv.CookieExpiration,
	}
}

func (ctx *Context) runMiddlewares(pr *PageRequest) *View {
	if ctx != pr.Context {
		panic("different Context in PageRequest")
	}
	for _, middleware := range ctx.middlewares {
		if view := middleware(pr); view != nil {
			return view
		}
	}
	return nil
}

// GetServiceLimiter returns the rate limiter for the given service and IP
func (ctx *Context) GetServiceLimiter(service, ip string) *rate.Limiter {
	if limiter, ok := ctx.Limiters[service]; ok {
		return limiter.Get(ip)
	}
	return nil
}

// ContextGetter ...
type ContextGetter func(context.Context, Layout) *Context
