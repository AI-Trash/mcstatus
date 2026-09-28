package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"mcstatus/internal/blocklist"
	"mcstatus/internal/cache"
	"mcstatus/internal/config"
	"mcstatus/internal/handlers"
	"mcstatus/internal/middleware"
)

type Server struct {
	cfg       *config.Config
	cache     *cache.Cache
	blocklist *blocklist.BlockList
	httpSrv   *http.Server
}

func New(cfg *config.Config) *Server {
	if cfg == nil {
		cfg = config.Load()
	}

	c := cache.New(cfg.CacheTTL, 10*time.Minute)
	bl := blocklist.New(cfg.MojangBlockedRefresh)

	mux := http.NewServeMux()

	// Java Status routes
	javaHandler := handlers.HandleJavaStatus(cfg, c, bl)
	mux.HandleFunc("GET /v2/status/java/{address...}", javaHandler)
	mux.HandleFunc("GET /status/java/{address...}", javaHandler)

	// Bedrock Status routes
	bedrockHandler := handlers.HandleBedrockStatus(cfg, c, bl)
	mux.HandleFunc("GET /v2/status/bedrock/{address...}", bedrockHandler)
	mux.HandleFunc("GET /status/bedrock/{address...}", bedrockHandler)

	// Icon routes
	iconHandler := handlers.HandleIcon(cfg, c)
	mux.HandleFunc("GET /v2/icon/{address...}", iconHandler)
	mux.HandleFunc("GET /v2/icon", iconHandler)
	mux.HandleFunc("GET /icon/{address...}", iconHandler)
	mux.HandleFunc("GET /icon", iconHandler)

	// Widget routes
	javaWidgetHandler := handlers.HandleJavaWidget(cfg, c, bl)
	mux.HandleFunc("GET /v2/widget/java/{address...}", javaWidgetHandler)
	mux.HandleFunc("GET /widget/java/{address...}", javaWidgetHandler)

	bedrockWidgetHandler := handlers.HandleBedrockWidget(cfg, c, bl)
	mux.HandleFunc("GET /v2/widget/bedrock/{address...}", bedrockWidgetHandler)
	mux.HandleFunc("GET /widget/bedrock/{address...}", bedrockWidgetHandler)

	// Vote routes
	voteHandler := handlers.HandleVote(cfg)
	mux.HandleFunc("POST /v2/vote", voteHandler)
	mux.HandleFunc("POST /vote", voteHandler)

	// Health check endpoints
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /healthz", healthHandler)

	// Root info endpoint
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := map[string]any{
			"service": "mcstatus",
			"status":  "operational",
			"docs":    "https://mcstatus.io/docs",
			"endpoints": []string{
				"/v2/status/java/{address}",
				"/v2/status/bedrock/{address}",
				"/v2/icon/{address}",
				"/v2/widget/java/{address}",
				"/v2/widget/bedrock/{address}",
				"/v2/vote",
			},
		}
		json.NewEncoder(w).Encode(info)
	})

	// Wrap with standard middlewares: Recovery -> Logging -> CORS
	handler := middleware.Recovery(middleware.Logging(middleware.CORS(mux)))

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Server{
		cfg:       cfg,
		cache:     c,
		blocklist: bl,
		httpSrv:   httpSrv,
	}
}

func (s *Server) Start() error {
	slog.Info("starting mcstatus server",
		"address", s.httpSrv.Addr,
		"cache_ttl", s.cfg.CacheTTL.String(),
	)
	err := s.httpSrv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down mcstatus server")
	s.cache.Close()
	s.blocklist.Close()
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.httpSrv.Handler
}
