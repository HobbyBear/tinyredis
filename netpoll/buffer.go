package netpoll

import (
	"fmt"
	"io"
	"math"
)

type RingBuffer struct {
	buf             []byte
	batchFetchBytes int64
	reader          io.Reader
	r               int // 标记下次读取开始的位置
	unReadSize      int // 缓冲区中未读数据大小
}

func NewRingBuffer(size int, rd io.Reader) *RingBuffer {
	return &RingBuffer{buf: make([]byte, size), batchFetchBytes: 1024, reader: rd}
}

func (r *RingBuffer) UnReadSize() int {
	return r.unReadSize
}

func (r *RingBuffer) fill() error {
	if r.unReadSize == len(r.buf) {
		return fmt.Errorf("the unReadSize is over range the buffer len")
	}
	readLen := int(math.Min(float64(r.batchFetchBytes), float64(len(r.buf)-r.unReadSize)))
	buf := make([]byte, readLen)
	readBytes, err := r.reader.Read(buf)
	if readBytes > 0 {
		end := r.r + r.unReadSize + readBytes
		if end < len(r.buf) {
			copy(r.buf[r.r+r.unReadSize:], buf[:readBytes])
		} else {
			writePos := (r.r + r.unReadSize) % len(r.buf)
			n := copy(r.buf[writePos:], buf[:readBytes])
			if n < readBytes {
				copy(r.buf[:end%len(r.buf)], buf[len(r.buf)-writePos:])
			}
		}
		r.unReadSize += readBytes
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func (r *RingBuffer) Peek(readOffsetBack, n int) ([]byte, error) {
	if n > len(r.buf) {
		return nil, fmt.Errorf("the unReadSize is over range the buffer len")
	}
peek:
	if n <= r.UnReadSize()-readOffsetBack {
		readPos := (r.r + readOffsetBack) % len(r.buf)
		return r.dataByPos(readPos, (r.r+readOffsetBack+n-1)%len(r.buf)), nil
	}
	err := r.fill()
	if err != nil {
		return nil, err
	}
	goto peek
}

// dataByPos 返回索引值在start和end之间的数据，闭区间
func (r *RingBuffer) dataByPos(start int, end int) []byte {
	// 因为环形缓冲区原因，所以末位置索引值有可能小于开始位置索引
	if end < start {
		return append(r.buf[start:], r.buf[:end+1]...)
	}
	return r.buf[start : end+1]
}

func (r *RingBuffer) PeekBytes(readOffsetBack int, delim byte) ([]byte, error) {
peek:
	for i := 0; i < r.unReadSize-readOffsetBack; i++ {
		if r.buf[(i+r.r+readOffsetBack)%len(r.buf)] == delim {
			return r.dataByPos((r.r+readOffsetBack)%len(r.buf), (i+r.r+readOffsetBack)%len(r.buf)), nil
		}
	}
	err := r.fill()
	if err != nil {
		return nil, err
	}
	goto peek
}

func (r *RingBuffer) AddReadPosition(n int) {
	r.r = (r.r + n) % len(r.buf)
	r.unReadSize -= n
}
