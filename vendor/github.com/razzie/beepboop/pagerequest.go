package beepboop

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mssola/user_agent"
	"github.com/razzie/babble"
	"github.com/razzie/reqip"
)

// PageRequest ...
type PageRequest struct {
	Context   *Context
	Request   *http.Request
	RequestID string
	RelPath   string
	RelURI    string
	PagePath  string
	Title     string
	IsAPI     bool
	renderer  LayoutRenderer
	logged    bool
	session   *Session
}

func newPageRequest(page *Page, r *http.Request, ctx *Context, renderer LayoutRenderer) *PageRequest {
	pr := &PageRequest{
		Context:   ctx,
		Request:   r,
		RequestID: newRequestID(),
		PagePath:  page.Path,
		Title:     page.Title,
		IsAPI:     renderer == nil,
		renderer:  renderer,
	}
	if pr.IsAPI {
		pr.RelPath = strings.TrimPrefix(r.URL.Path, "/api"+page.Path)
		pr.RelURI = strings.TrimPrefix(r.URL.RequestURI(), "/api"+page.Path)
	} else {
		pr.RelPath = strings.TrimPrefix(r.URL.Path, page.Path)
		pr.RelURI = strings.TrimPrefix(r.URL.RequestURI(), page.Path)
	}
	return pr
}

func newRequestID() string {
	i := uint16(time.Now().UnixNano())
	babbler := babble.NewBabbler()
	return fmt.Sprintf("%s-%x", babbler.Babble(), i)
}

func (r *PageRequest) logRequest() {
	ip := reqip.GetClientIP(r.Request)
	ua := user_agent.New(r.Request.UserAgent())
	browser, ver := ua.Browser()

	logmsg := fmt.Sprintf("[%s]: %s %s\n • %s, %s %s %s",
		r.RequestID, r.Request.Method, r.Request.RequestURI,
		ip, ua.OS(), browser, ver)

	var hasLocation bool
	if r.Context.GeoIPClient != nil {
		loc, _ := r.Context.GeoIPClient.GetLocation(context.Background(), ip)
		if loc != nil {
			hasLocation = true
			logmsg += ", " + loc.String()
		}
	}
	if !hasLocation {
		hostnames, _ := net.LookupAddr(ip)
		logmsg += ", " + strings.Join(hostnames, ", ")
	}

	session, _ := r.Request.Cookie("session")
	if session != nil {
		logmsg += ", session: " + session.Value
	}

	r.Context.Logger.Print(logmsg)
	r.logged = true
}

func (r *PageRequest) logRequestNonblocking() {
	r.logged = true
	go r.logRequest()
}

// Log ...
func (r *PageRequest) Log(a ...interface{}) {
	if !r.logged {
		r.logRequestNonblocking()
	}
	prefix := fmt.Sprintf("[%s] ", r.RequestID)
	r.Context.Logger.Output(2, prefix+fmt.Sprint(a...))
}

// Logf ...
func (r *PageRequest) Logf(format string, a ...interface{}) {
	if !r.logged {
		r.logRequestNonblocking()
	}
	prefix := fmt.Sprintf("[%s] ", r.RequestID)
	r.Context.Logger.Output(2, prefix+fmt.Sprintf(format, a...))
}

// Respond returns the default page response View
func (r *PageRequest) Respond(data interface{}, opts ...ViewOption) *View {
	v := &View{
		StatusCode: http.StatusOK,
		Data:       data,
	}
	for _, opt := range opts {
		opt(v)
	}
	v.renderer = func(w http.ResponseWriter) {
		r.renderer(w, r.Request, r.Title, data, v.StatusCode)
	}
	return v
}

// Session returns the current session
func (r *PageRequest) Session() *Session {
	if r.session == nil {
		r.session = newSession(r)
	}
	return r.session
}

func (r *PageRequest) updateSession(view *View) {
	if r.session != nil {
		view.cookies = append(view.cookies, r.session.toCookies(r.Context.CookieExpiration)...)
	}
}
