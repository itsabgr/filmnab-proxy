package main

import (
	"context"
	"crypto/ed25519"
	"mime"
	"net/http"
	"path/filepath"
	"time"
)

var _ http.Handler = (*Server)(nil)

type Server struct {
	publicKeys              []ed25519.PublicKey
	cache                   iCache
	corsHeader, cacheHeader string
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Robots-Tag", "noindex, nofollow")
	_ = request.Body.Close()
	ctx, cancel := context.WithTimeout(request.Context(), time.Second*10)
	defer cancel()
	request = request.WithContext(ctx)
	writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET")
	writer.Header().Set("Access-Control-Allow-Origin", s.corsHeader)
	switch request.Method {
	case http.MethodGet:
	case http.MethodOptions:
		writer.WriteHeader(http.StatusNoContent)
		return
	default:
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Cache-Control", s.cacheHeader)
	filePath, err := Auth(request.URL.Path, s.publicKeys...)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusUnauthorized)
		return
	}
	res, err := s.cache.Get(request.Context(), filePath)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Add("X-Cache", res.Header())
	if len(res.Value) == 0 {
		http.NotFound(writer, request)
		return
	}
	if ext := filepath.Ext(filePath); ext != "" {
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			writer.Header().Add("Content-Type", mimeType)
		}
	}
	_, _ = writer.Write(res.Value)
}
