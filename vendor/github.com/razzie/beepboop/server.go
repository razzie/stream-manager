package beepboop

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	geoclient "github.com/razzie/geoip-server/client"
	"github.com/razzie/geoip-server/geoip"
)

// Server ...
type Server struct {
	mux              http.ServeMux
	Layout           Layout
	FaviconPNG       []byte
	Header           http.Header
	Metadata         map[string]string
	DB               *DB
	Logger           *log.Logger
	GeoIPClient      geoip.Client
	Limiters         map[string]*RateLimiter
	Middlewares      []Middleware
	CookieExpiration time.Duration
}

// NewServer creates a new Server
func NewServer() *Server {
	srv := &Server{
		Layout:           DefaultLayout,
		FaviconPNG:       favicon,
		Header:           map[string][]string{"Server": {"beepboop"}},
		Metadata:         map[string]string{"generator": "https://github.com/razzie/beepboop"},
		Logger:           log.New(os.Stdout, "", log.LstdFlags),
		GeoIPClient:      geoclient.DefaultClient,
		Limiters:         make(map[string]*RateLimiter),
		CookieExpiration: time.Hour * 24 * 7,
	}
	srv.mux.HandleFunc("/favicon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(srv.FaviconPNG)))
		_, _ = w.Write(srv.FaviconPNG)
	})
	srv.mux.Handle("/favicon.ico", http.RedirectHandler("/favicon.png", http.StatusMovedPermanently))
	return srv
}

// AddPage adds a new servable page to the server
func (srv *Server) AddPage(page *Page) error {
	return srv.AddPageWithLayout(page, srv.Layout)
}

// AddPageWithLayout adds a new servable page with custom layout to the server
func (srv *Server) AddPageWithLayout(page *Page, layout Layout) error {
	page.addMetadata(srv.Metadata)
	renderer, err := page.GetHandler(layout, srv.getContext)
	if err != nil {
		return err
	}

	srv.mux.Handle(page.Path, renderer)
	srv.mux.Handle("/api"+page.Path, page.GetAPIHandler(srv.getContext))
	return nil
}

// AddPages adds multiple pages to the server and panics if anything goes wrong
func (srv *Server) AddPages(pages ...*Page) {
	for _, page := range pages {
		err := srv.AddPage(page)
		if err != nil {
			panic(err)
		}
	}
}

// AddServiceRate limit sets up a rate limiter for a given service name
// which can be used by page handlers and middlewares
func (srv *Server) AddServiceRate(service string, interval time.Duration, n int) {
	srv.Limiters[service] = NewRateLimiter(interval, n)
}

// AddMiddleware adds a middleware
func (srv *Server) AddMiddleware(middleware Middleware) {
	srv.Middlewares = append(srv.Middlewares, middleware)
}

// AddMiddlewares adds middlewares
func (srv *Server) AddMiddlewares(middlewares ...Middleware) {
	srv.Middlewares = append(srv.Middlewares, middlewares...)
}

// ConnectDB ...
func (srv *Server) ConnectDB(redisUrl string) error {
	db, err := NewDB(redisUrl)
	if err != nil {
		return err
	}

	srv.DB = db
	return nil
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	for key, values := range srv.Header {
		key = http.CanonicalHeaderKey(key)
		h[key] = append(h[key], values...)
	}
	srv.mux.ServeHTTP(w, r)
}

func (srv *Server) getContext(ctx context.Context, layout Layout) *Context {
	return newContext(ctx, layout, srv)
}

var favicon, _ = base64.StdEncoding.DecodeString("" +
	"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" +
	"ciHyOgH8D7fj0G91jES5AAAAAElFTkSuQmCC")
