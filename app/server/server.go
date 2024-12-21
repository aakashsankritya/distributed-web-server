package server

import (
	"coding/constants"
	"coding/logger"
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

var (
	serverOnce sync.Once
	ctxOnce    sync.Once
	ctx        context.Context
	cancelFunc context.CancelFunc
	webServer  *WebServer
)

type Request struct {
	Id       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

type WebServer struct {
	logger             *logrus.Logger
	aggregatorLog      *logrus.Logger
	redisClient        *redis.Client
	wg                 *sync.WaitGroup
	ctx                context.Context
	cancelFunc         context.CancelFunc
	reqProcessors      []chan *Request
	endpointProcessors []chan *Request
	config             *constants.Config
	kafkaProducer      *kafka.Writer
}

func NewWebServer(logFilePath, redisAddress, kafkaUrl, configFilePath string) *WebServer {
	ctx, cancelFunc := GetContext()

	serverOnce.Do(func() {
		config, _ := constants.LoadConfig(configFilePath)
		webServer = &WebServer{
			config:             config,
			wg:                 &sync.WaitGroup{},
			ctx:                ctx,
			cancelFunc:         cancelFunc,
			logger:             logger.GetLogger(logFilePath, true),
			redisClient:        GetRedisClient(redisAddress),
			aggregatorLog:      logger.GetLogger(config.AGGREGATED_LOG_FILE_NAME, false),
			reqProcessors:      make([]chan *Request, config.WORKER_POOL_SIZE),
			endpointProcessors: make([]chan *Request, config.WORKER_POOL_SIZE),
			kafkaProducer:      kafka.NewWriter(kafka.WriterConfig{Brokers: []string{kafkaUrl}, Topic: config.KAFKA_TOPIC, Balancer: &kafka.LeastBytes{}}),
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
	server := &http.Server{
		Addr:         port,
		Handler:      s.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.wg.Add(3)
	s.initWorkerPool()
	go s.startWorkerPool()
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

func (s *WebServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/verve/accept", s.handleAccept)
	return mux
}

func (s *WebServer) handleAccept(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	endpoint := r.URL.Query().Get("endpoint")
	if id == "" {
		http.Error(w, "failed", http.StatusBadRequest)
		return
	}
	workerId := s.hashToWorkerId(id)
	request := &Request{Id: id, Endpoint: endpoint}
	s.reqProcessors[workerId] <- request
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *WebServer) hashToWorkerId(id string) int {
	return int(len(id) % s.config.WORKER_POOL_SIZE)
}

func (s *WebServer) waitForShutdown(server *http.Server) {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan
	s.stopWorkerPool()
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
