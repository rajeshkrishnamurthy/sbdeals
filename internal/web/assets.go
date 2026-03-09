package web

import (
	_ "embed"
	"net/http"
)

//go:embed assets/books-form.js
var booksFormJS []byte

//go:embed assets/bundles-form.js
var bundlesFormJS []byte

//go:embed assets/rails-form.js
var railsFormJS []byte

//go:embed assets/catalog.js
var catalogJS []byte

//go:embed assets/enquiries-form.js
var enquiriesFormJS []byte

func (s *Server) handleBooksFormJSAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(booksFormJS)
}

func (s *Server) handleBundlesFormJSAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(bundlesFormJS)
}

func (s *Server) handleRailsFormJSAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(railsFormJS)
}

func (s *Server) handleCatalogJSAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(catalogJS)
}

func (s *Server) handleEnquiriesFormJSAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(enquiriesFormJS)
}
