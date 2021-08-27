package log

import "sync"

type (
	LogOutput struct {
		Type  uint8
		Bytes []byte
	}
	StreamWatchers struct {
		*sync.RWMutex
		watchers map[chan LogOutput]bool
	}
)

func NewStreamWatcher() *StreamWatchers {
	return &StreamWatchers{
		RWMutex:  &sync.RWMutex{},
		watchers: make(map[chan LogOutput]bool),
	}
}

// addWatcher adds the provided channel to the streamWatchers set
func (s *StreamWatchers) AddWatcher(c chan LogOutput) {
	s.Lock()
	defer s.Unlock()

	s.watchers[c] = true
}

// sendLog sends the log message to all the subscribers in the streamWatchers set
func (s *StreamWatchers) SendLog(log LogOutput) {
	s.RLock()
	defer s.RUnlock()

	for c := range s.watchers {
		c <- log
	}
}

// removeLogWatcher removes the channel from the logWatchers set
func (s *StreamWatchers) RemoveLogWatcher(c chan LogOutput) {
	s.Lock()
	defer s.Unlock()

	delete(s.watchers, c)
}
