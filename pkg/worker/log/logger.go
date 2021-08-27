package log

type Stream struct {
	streamChan chan []byte
}

func NewStreamOutput(streamChan chan []byte) *Stream {
	s := &Stream{
		streamChan: streamChan,
	}
	return s
}

func (s *Stream) Write(b []byte) (int, error) {
	n := len(b)

	buf := make([]byte, len(b))
	copy(buf, b)

	s.streamChan <- buf

	return n, nil
}

func (s *Stream) Bytes() chan []byte {
	return s.streamChan

}
