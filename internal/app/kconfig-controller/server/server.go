package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/klog"
)

const (
	healthEndpointPath  = "/health"
	metricsEndpointPath = "/prometheus"

	contentTypeKey  = "Content-Type"
	jsonContentType = "application/json"
	textContentType = "text/plain"

	invalidRequestError = "InvalidRequest"
)

// Server for serving health checks, metrics, profiling, etc.
type Server struct {
	cfg        Cfg
	httpserver *http.Server
	router     *mux.Router
}

// Cfg configuration object for Server
type Cfg struct {
	Port int
}

// NewServer creates new server
func NewServer(cfg Cfg) *Server {
	server := &Server{
		cfg:    cfg,
		router: mux.NewRouter(),
	}
	server.setHandlers()
	server.httpserver = &http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: server.router}
	return server
}

// Start run server
func (s *Server) Start() {
	klog.Infof("Starting server - http port %d", s.cfg.Port)
	if err := s.httpserver.ListenAndServe(); err != nil {
		klog.Fatalf("Failed to start server: %s", err.Error())
	}
}

// Stop stop server for shutdown
func (s *Server) Stop() {
	klog.Infof("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.httpserver.Shutdown(ctx)
}

func (s *Server) setHandlers() {
	// HealthCheck
	s.router.HandleFunc(healthEndpointPath, s.Health).Methods(http.MethodGet)
}

// Health simple ping response. Healthy as long as server is running.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	status := struct {
		Status string `json:"status"`
	}{}
	status.Status = "OK"
	w.WriteHeader(http.StatusOK)
	w.Header().Set(contentTypeKey, jsonContentType)
	json.NewEncoder(w).Encode(status)
}
