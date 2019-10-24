package buffer

import (
	"log"
	"sync"
)

// WaveSource は、ソース波形の供給用バッファです。
type WaveSource struct {
	index  int
	buffer []float64
	notify chan struct{}
	mutex  sync.Mutex
}

// NewWaveSource は、新しい WaveSource を作成します。
func NewWaveSource() *WaveSource {
	return &WaveSource{
		notify: make(chan struct{}, 1),
	}
}

// Append は、ソース波形をバッファに蓄積します。
func (s *WaveSource) Append(data []float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	c0 := cap(s.buffer)
	s.buffer = append(s.buffer, data...)
	c1 := cap(s.buffer)
	if len(s.notify) == 0 {
		s.notify <- struct{}{}
	}
	if c1 < c0 {
		log.Printf("debug: buffer reallocated %d -> %d", c0, c1)
	}
}

// Close は、ソース波形の供給を終了します。
func (s *WaveSource) Close() {
	close(s.notify)
}

func (s *WaveSource) readAsync(begin, end int) ([]float64, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	begin -= s.index
	if begin < 0 {
		return nil, false
	}
	end -= s.index
	if len(s.buffer) < end {
		return s.buffer[begin:], false
	}
	return s.buffer[begin:end], true
}

// Read は、指定した範囲のソース波形を取得します。
// 供給が追いついていない場合はブロックします。
// 供給ソースがクローズした場合は、第2の返り値が false となります。
// このとき、第1の返り値は長さ (end-begin) に満たない場合があります。
func (s *WaveSource) Read(begin, end int) ([]float64, bool) {
	for {
		data, ok := s.readAsync(begin, end)
		if ok {
			return data, true
		}
		_, ok = <-s.notify
		if !ok {
			break
		}
	}
	data, _ := s.readAsync(begin, end)
	return data, false
}

// DiscardUntil は、指定した位置以前のバッファを破棄します。
func (s *WaveSource) DiscardUntil(i int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	d := i - s.index
	s.index = i
	if d <= 0 {
		return
	}
	if len(s.buffer) <= d {
		s.buffer = nil
		log.Printf("debug: buffer empty")
		return
	}
	s.buffer = s.buffer[d:]
}
