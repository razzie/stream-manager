package beepboop

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// View is something that a PageHandler returns and is capable of rendering a page
type View struct {
	StatusCode int
	Error      error
	Data       interface{}
	Redirect   string
	header     http.Header
	cookies    []*http.Cookie
	renderer   func(w http.ResponseWriter)
	closer     func() error
}

// Render renders the view
func (view *View) Render(w http.ResponseWriter) {
	h := w.Header()
	for key, values := range view.header {
		key = http.CanonicalHeaderKey(key)
		h[key] = append(h[key], values...)
	}
	for _, cookie := range view.cookies {
		http.SetCookie(w, cookie)
	}
	view.renderer(w)
}

// RenderAPIResponse renders the API response of the view
func (view *View) RenderAPIResponse(w http.ResponseWriter) {
	h := w.Header()
	for key, values := range view.header {
		key = http.CanonicalHeaderKey(key)
		h[key] = append(h[key], values...)
	}
	for _, cookie := range view.cookies {
		http.SetCookie(w, cookie)
	}
	w.WriteHeader(view.StatusCode)

	if view.Error != nil {
		w.Write([]byte(view.Error.Error()))
		return
	}

	if view.Data != nil {
		data, err := json.MarshalIndent(view.Data, "", "\t")
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(data)
		return
	}

	if view.StatusCode == http.StatusOK {
		w.Write([]byte("OK"))
		return
	}

	w.Write([]byte(http.StatusText(view.StatusCode)))
}

// Close frees resources used by the view
func (view *View) Close() error {
	if view.closer != nil {
		return view.closer()
	}
	return nil
}

// ViewOption is used to customize the error message, error code or data in the view
type ViewOption func(view *View)

// WithError sets the view error and error code
func WithError(err error, errcode int) ViewOption {
	return func(view *View) {
		view.Error = err
		view.StatusCode = errcode
	}
}

// WithErrorMessage sets the view error message and error code
func WithErrorMessage(errmsg string, errcode int) ViewOption {
	return WithError(fmt.Errorf("%s", errmsg), errcode)
}

// WithData sets the view data
func WithData(data interface{}) ViewOption {
	return func(view *View) {
		view.Data = data
	}
}

// WithHeader adds a header field to the view
func WithHeader(key, value string) ViewOption {
	return func(view *View) {
		if view.header == nil {
			view.header = make(http.Header)
		}
		view.header.Add(key, value)
	}
}

// WithCookie adds a cookie to the view
func WithCookie(cookie *http.Cookie) ViewOption {
	return func(view *View) {
		view.cookies = append(view.cookies, cookie)
	}
}

var defaultErrorRenderer = GetErrorRenderer(DefaultLayout)

// CustomErrorView returns a View that represents an error and uses a custom renderer
func CustomErrorView(r *http.Request, errmsg string, errcode int, renderer ErrorRenderer, opts ...ViewOption) *View {
	v := &View{
		StatusCode: errcode,
		Error:      fmt.Errorf("%s", errmsg),
	}
	for _, opt := range opts {
		opt(v)
	}
	v.renderer = func(w http.ResponseWriter) {
		renderer(w, r, errmsg, v.StatusCode)
	}
	return v
}

// CustomErrorView returns a View that represents an error and uses a custom renderer
func (r *PageRequest) CustomErrorView(errmsg string, errcode int, renderer ErrorRenderer, opts ...ViewOption) *View {
	return CustomErrorView(r.Request, errmsg, errcode, renderer, opts...)
}

// ErrorView returns a View that represents an error
func ErrorView(r *http.Request, errmsg string, errcode int, opts ...ViewOption) *View {
	return CustomErrorView(r, errmsg, errcode, defaultErrorRenderer, opts...)
}

// ErrorView returns a View that represents an error
func (r *PageRequest) ErrorView(errmsg string, errcode int, opts ...ViewOption) *View {
	return ErrorView(r.Request, errmsg, errcode, opts...)
}

// EmbedView returns a View that embeds the given URL
func EmbedView(url string, opts ...ViewOption) *View {
	v := &View{
		StatusCode: http.StatusOK,
		Data:       url,
	}
	for _, opt := range opts {
		opt(v)
	}
	v.renderer = func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(v.StatusCode)
		fmt.Fprintf(w, `<iframe src="%s" style="position:fixed; top:0; left:0; bottom:0; right:0; width:100%%; height:100%%; border:none; margin:0; padding:0; overflow:hidden; z-index:999999;"></iframe>`, url)
	}
	return v
}

// EmbedView returns a View that embeds the given URL
func (r *PageRequest) EmbedView(url string, opts ...ViewOption) *View {
	return EmbedView(url, opts...)
}

// RedirectView returns a View that redirects to the given URL
func RedirectView(r *http.Request, url string, opts ...ViewOption) *View {
	v := &View{
		StatusCode: http.StatusOK,
		Redirect:   url,
	}
	for _, opt := range opts {
		opt(v)
	}
	v.renderer = func(w http.ResponseWriter) {
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
	return v
}

// RedirectView returns a View that redirects to the given URL
func (r *PageRequest) RedirectView(url string, opts ...ViewOption) *View {
	return RedirectView(r.Request, url, opts...)
}

// CopyView returns a View that copies the content of a http.Response
func CopyView(resp *http.Response, opts ...ViewOption) *View {
	v := &View{
		StatusCode: resp.StatusCode,
		Data:       resp,
	}
	for _, opt := range opts {
		opt(v)
	}
	bytes, _ := ioutil.ReadAll(resp.Body)
	v.renderer = func(w http.ResponseWriter) {
		for k, v := range resp.Header {
			w.Header().Set(k, v[0])
		}
		w.WriteHeader(v.StatusCode)
		w.Write(bytes)
	}
	return v
}

// CopyView returns a View that copies the content of a http.Response
func (r *PageRequest) CopyView(resp *http.Response, opts ...ViewOption) *View {
	return CopyView(resp, opts...)
}

// AsyncCopyView returns a View that copies the content of a http.Response asynchronously
func AsyncCopyView(resp *http.Response, opts ...ViewOption) *View {
	v := &View{
		StatusCode: resp.StatusCode,
		Data:       resp,
		closer:     resp.Body.Close,
	}
	for _, opt := range opts {
		opt(v)
	}
	v.renderer = func(w http.ResponseWriter) {
		for k, v := range resp.Header {
			w.Header().Set(k, v[0])
		}
		w.WriteHeader(v.StatusCode)
		io.Copy(w, resp.Body)
	}
	return v
}

// AsyncCopyView returns a View that copies the content of a http.Response asynchronously
func (r *PageRequest) AsyncCopyView(resp *http.Response, opts ...ViewOption) *View {
	return AsyncCopyView(resp, opts...)
}

// HandlerView returns a View that uses a http.HandlerFunc to render a response
func HandlerView(r *http.Request, handler http.HandlerFunc, opts ...ViewOption) *View {
	v := &View{
		StatusCode: http.StatusOK,
	}
	for _, opt := range opts {
		opt(v)
	}
	v.renderer = func(w http.ResponseWriter) {
		handler(w, r)
	}
	return v
}

// HandlerView returns a View that uses a http.HandlerFunc to render a response
func (r *PageRequest) HandlerView(handler http.HandlerFunc, opts ...ViewOption) *View {
	return HandlerView(r.Request, handler, opts...)
}

// FileView returns a View that serves a file
func FileView(r *http.Request, file http.File, mime string, attachment bool, opts ...ViewOption) *View {
	v := &View{
		StatusCode: http.StatusOK,
		closer:     file.Close,
	}
	for _, opt := range opts {
		opt(v)
	}
	fi, err := file.Stat()
	if err != nil {
		v.Error = err
		v.renderer = func(w http.ResponseWriter) {
			defaultErrorRenderer(w, r, err.Error(), http.StatusInternalServerError)
		}
		return v
	}
	v.renderer = func(w http.ResponseWriter) {
		if attachment {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fi.Name()))
		}
		if len(mime) > 0 {
			w.Header().Set("Content-Type", mime)
		}
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), file)
	}
	return v
}

// FileView returns a View that serves a file
func (r *PageRequest) FileView(file http.File, mime string, attachment bool, opts ...ViewOption) *View {
	return FileView(r.Request, file, mime, attachment, opts...)
}
