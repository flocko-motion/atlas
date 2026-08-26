// package: rest_http / transport
// type:    adapter
// job:     serves explorer.html when the `explorer` build tag embedded it (-> frontend)
// limits:  a static asset route, no API surface — not part of the OpenAPI contract
package rest_http

import (
	"net/http"

	"github.com/flocko-motion/rankedb/frontend"
)

// explorerHandler serves the embedded explorer.html when enabled allows it, 404
// otherwise — either because config's "explorer" said not to, or because this binary
// was built without the `explorer` tag (frontend.Explorer is then an empty embed.FS).
func explorerHandler(enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			http.Error(w, "explorer disabled by config", http.StatusNotFound)
			return
		}
		b, err := frontend.Explorer.ReadFile("dist/explorer.html")
		if err != nil {
			http.Error(w, "explorer not embedded in this build — built without -tags explorer", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	}
}
