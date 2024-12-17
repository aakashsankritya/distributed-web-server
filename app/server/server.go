package server

import (
	"coding/logger"
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

type WebServer struct {
	logger      *logrus.Logger
	redisClient *redis.Client
	wg          sync.WaitGroup
}

func NewWebServer(logFilePath, redisAddress string) *WebServer {
	server := &WebServer{}
	server.logger = logger.GetLogger(logFilePath)
	server.redisClient = GetRedisClient(redisAddress)
	return server
}

func GetRedisClient(redisAddress string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: redisAddress,
		DB:   0,
	})
}

func (s *WebServer) Start(port string) {
	// go s.aggregateAndLog()
	server := &http.Server{Addr: port, Handler: s.routes()}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Infof("Server is running on port %s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Errorf("HTTP server error: %v", err)
		}
	}()
	// Wait for shutdown signal
	s.waitForShutdown(server)
}

func (s *WebServer) handleAccept(w http.ResponseWriter, r *http.Request) {
	// ctx := context.Background()
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	// Add ID to Redis set
	// if err := s.redisClient.SAdd(ctx, s.redisKey, id).Err(); err != nil {
	// 	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 	log.Printf("Failed to add ID to Redis: %v", err)
	// 	return
	// }

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *WebServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/accept", s.handleAccept)
	return mux
}

func (s *WebServer) waitForShutdown(server *http.Server) {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan
	s.logger.Println("Received termination signal, shutting down gracefully...")
	// Cancel context to stop background tasks
	// s.cancel()

	// Shutdown HTTP server
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShutDown); err != nil {
		s.logger.Fatalf("Server shutdown failed: %v", err)
	}
	s.logger.Println("HTTP server shutdown successfully")
	// Wait for all goroutines to finish
	s.wg.Wait()
}
