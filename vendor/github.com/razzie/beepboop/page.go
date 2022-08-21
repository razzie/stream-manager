package beepboop

import (
	"io/fs"
	"net/http"
	"os"
	"path"
)

// Page ...
type Page struct {
	Path            string
	Title           string
	ContentTemplate string
	Stylesheets     []string
	Scripts         []string
	Metadata        map[string]string
	Handler         func(*PageRequest) *View
	OnlyLogOnError  bool
}

// GetHandler creates a http.Handler that uses the given layout to render the page
func (page *Page) GetHandler(layout Layout, getctx ContextGetter) (http.Handler, error) {
	renderer, err := layout.BindTemplate(page.ContentTemplate, page.Stylesheets, page.Scripts, page.Metadata)
	if err != nil {
		return nil, err
	}
	return page.getHandler(getctx, layout, renderer), nil
}

// GetAPIHandler creates a http.Handler that handles API requests of the page
func (page *Page) GetAPIHandler(getctx ContextGetter) http.Handler {
	return page.getHandler(getctx, nil, nil)
}

func (page *Page) getHandler(getctx ContextGetter, layout Layout, renderer LayoutRenderer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := getctx(r.Context(), layout)
		pr := newPageRequest(page, r, ctx, renderer)
		if !page.OnlyLogOnError {
			pr.logRequestNonblocking()
		}

		view := ctx.runMiddlewares(pr)
		if view == nil && page.Handler != nil {
			view = page.Handler(pr)
		}
		if view == nil {
			view = pr.Respond(nil)
		}

		defer view.Close()
		pr.updateSession(view)
		if pr.IsAPI {
			view.RenderAPIResponse(w)
		} else {
			view.Render(w)
		}
	})
}

func (page *Page) addMetadata(meta map[string]string) {
	if page.Metadata == nil && len(meta) > 0 {
		page.Metadata = make(map[string]string)
	}
	for name, content := range meta {
		page.Metadata[name] = content
	}
}

// StaticAssetPage returns a page that serves static assets from a directory
func StaticAssetPage(pagePath, assetDir string) *Page {
	handler := func(pr *PageRequest) *View {
		uri := path.Clean(pr.RelPath)
		if fi, _ := os.Stat(path.Join(assetDir, uri)); fi != nil && fi.IsDir() {
			return pr.ErrorView("Forbidden", http.StatusForbidden)
		}
		return pr.HandlerView(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, path.Join(assetDir, uri))
		})
	}
	return &Page{
		Path:           pagePath,
		Handler:        handler,
		OnlyLogOnError: true,
	}
}

// FSPage returns a page that serves assets from fs.FS
func FSPage(pagePath string, f fs.FS) *Page {
	handler := func(pr *PageRequest) *View {
		return pr.HandlerView(http.FileServer(http.FS(f)).ServeHTTP)
	}
	return &Page{
		Path:           pagePath,
		Handler:        handler,
		OnlyLogOnError: true,
	}
}
