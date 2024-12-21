package server

import (
	"fmt"
	"sync"
)

func (s *WebServer) initWorkerPool() {
	for i := 0; i < s.config.WORKER_POOL_SIZE; i++ {
		s.reqProcessors[i] = make(chan *Request, s.config.WORKER_CHANNEL_BUFFER_SIZE)
	}
}

func (s *WebServer) startWorkerPool() {
	defer s.wg.Done()
	for i := 0; i < len(s.reqProcessors); i++ {
		s.wg.Add(1)
		go s.processChannel(s.wg, i)
	}
}

func (s *WebServer) processChannel(wg *sync.WaitGroup, channelId int) {
	defer wg.Done()
	wg.Add(2)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for req := range s.reqProcessors[channelId] {
			_, err := s.redisClient.SAdd(s.config.REQUEST_SET_KEY, req.Id).Result()
			if err != nil {
				s.logger.Errorf("failed adding key to redis for channelId: %d, error: %v", channelId, err)
				return
			}

			if req.Endpoint != "" {
				s.addEndpoint(req)
			}
		}
	}(wg)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for req := range s.reqProcessors[channelId] {
			if req.Endpoint != "" {
				uniqueReqCount, err := s.redisClient.SCard(s.config.REQUEST_SET_KEY).Result()
				if err != nil {
					s.logger.Errorf("failed to fetch count of unique request from redis: %v", err)
					return
				}
				url := fmt.Sprintf("%s?count=%d", req.Endpoint, uniqueReqCount)
				s.callExternalAPI(url, "POST", []byte{})
			}
		}
	}(wg)
}

// since calling these endpoint could be blocking, need to use separate goroutines
func (s *WebServer) addEndpoint(req *Request) {
	endpointProcessorId := s.hashToWorkerId(req.Id)
	s.endpointProcessors[endpointProcessorId] <- req
}

func (s *WebServer) stopWorkerPool() {
	for i := 0; i < len(s.reqProcessors); i++ {
		close(s.reqProcessors[i])
		close(s.endpointProcessors[i])
	}
}
