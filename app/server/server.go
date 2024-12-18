package server

import (
	"bufio"
	"coding/logger"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

var (
	serverOnce sync.Once
	ctxOnce    sync.Once
	ctx        context.Context
	cancelFunc context.CancelFunc
	webServer  *WebServer
)

type WebServer struct {
	logger      *logrus.Logger
	redisClient *redis.Client
	wg          sync.WaitGroup
	ctx         context.Context
	cancelFunc  context.CancelFunc
}

func NewWebServer(logFilePath, redisAddress string) *WebServer {
	ctx, cancelFunc := GetContext()
	serverOnce.Do(func() {
		webServer = &WebServer{
			wg:          sync.WaitGroup{},
			ctx:         ctx,
			cancelFunc:  cancelFunc,
			logger:      logger.GetLogger(logFilePath),
			redisClient: GetRedisClient(redisAddress),
		}
	})
	return webServer
}

func GetContext() (context.Context, context.CancelFunc) {
	ctxOnce.Do(func() {
		ctx, cancelFunc = context.WithCancel(context.Background())
	})
	return ctx, cancelFunc
}

func GetRedisClient(redisAddress string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: redisAddress,
		DB:   0,
	})
}

func (s *WebServer) Start(port string) {
	server := &http.Server{Addr: port, Handler: s.routes()}
	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		s.logger.Infof("Server is running on port %s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Errorf("HTTP server error: %v", err)
		}
	}()
	go s.StartAggregator()
	// Wait for shutdown signal
	s.waitForShutdown(server)
}

func (s *WebServer) handleAccept(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	endpoint := r.URL.Query().Get("endpoint")
	if id == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}
	redisKey := fmt.Sprintf("REQ_ID:%s", id)
	err := s.redisClient.Incr(redisKey).Err()
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal Server Error: Failed to process request id: %s", id), http.StatusInternalServerError)
	}
	if endpoint != "" {
		// TODO:
		// Call to external endpoint
		s.logger.Warnf("expected to call external endpoint %s", endpoint)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *WebServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/verve/accept", s.handleAccept)
	return mux
}

func (s *WebServer) waitForShutdown(server *http.Server) {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan
	s.logger.Println("Received termination signal, shutting down gracefully...")
	// Cancel context to stop background tasks
	s.cancelFunc()
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

func (s *WebServer) StartAggregator() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			err := s.aggregateAndCleanRedis()
			if err != nil {
				s.logger.Errorf("Error during aggregation: %v", err)
			}
		}
	}
}

func (s *WebServer) aggregateAndCleanRedis() error {
	acquired, err := s.redisClient.SetNX("AGGREGATOR_LOCK", true, 1*time.Minute).Result()
	if err != nil {
		return err
	}
	if !acquired {
		s.logger.Info("aggregation step is processed by other node")
		return nil
	}
	keys, err := s.redisClient.Keys("REQ_ID:*").Result()
	if err != nil {
		return fmt.Errorf("error fetching keys from redis: %w", err)
	}
	pipe := s.redisClient.Pipeline()
	counts := make(map[string]int64)
	for _, key := range keys {
		pipe.Get(key)
	}
	cmnds, err := pipe.Exec()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("error executing pipeline: %w", err)
	}

	for i, cmnd := range cmnds {
		val, err := cmnd.(*redis.StringCmd).Int64()
		if err == nil {
			counts[keys[i]] = val
		}
	}
	s.writeToLogFile(counts)
	return nil
}

func (s *WebServer) writeToLogFile(data map[string]int64) {
	file, err := os.OpenFile("logs/aggregated_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		s.logger.Errorf("Error opening log file: %v", err)
		return
	}
	defer file.Close()

	timestamp := time.Now().Format(time.RFC3339)
	writer := bufio.NewWriter(file)
	fmt.Fprintf(writer, "\nTimestamp: %s\n", timestamp)
	fmt.Fprintf(writer, "Aggregated Data:\n")
	for id, count := range data {
		fmt.Fprintf(writer, "\tID: %s, Count: %d\n", id, count)
	}
	writer.Flush()
}
