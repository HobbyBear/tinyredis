package netpoll

import (
	"fmt"
	"io"
	"math"
	"tinyredis/log"
)

type RingBuffer struct {
	buf             []byte
	batchFetchBytes int64
	reader          io.Reader
	r               int // 标记下次读取开始的位置
	unReadSize      int // 缓冲区中未读数据大小
}

func NewRingBuffer(size int, rd io.Reader) *RingBuffer {
	return &RingBuffer{buf: make([]byte, size), batchFetchBytes: 10, reader: rd}
}

func (b *RingBuffer) UnReadSize() int {
	return b.unReadSize
}

func (b *RingBuffer) Peek(readOffset, n int) ([]byte, error) {
	if n > len(b.buf) {
		return nil, fmt.Errorf("the unReadSize is over range the buffer len")
	}
peek:
	if n <= b.UnReadSize()-readOffset {
		readPos := (b.r + readOffset) % len(b.buf)
		return b.DataByPos(readPos, (b.r+readOffset+n-1)%len(b.buf)), nil
	}
	if b.unReadSize == len(b.buf) {
		return nil, fmt.Errorf("the unReadSize is over range the buffer len")
	}
	readLen := int(math.Min(float64(b.batchFetchBytes), float64(len(b.buf)-b.unReadSize)))
	buf := make([]byte, readLen)
	readBytes, err := b.reader.Read(buf)
	if readBytes > 0 {
		end := b.r + b.unReadSize + readBytes
		if end < len(b.buf) {
			copy(b.buf[b.r+b.unReadSize:], buf[:readBytes])
		} else {
			writePos := (b.r + b.unReadSize) % len(b.buf)
			n := copy(b.buf[writePos:], buf[:readBytes])
			if n < readBytes {
				copy(b.buf[:end%len(b.buf)], buf[len(b.buf)-writePos:])
			}
		}
		b.unReadSize += readBytes
		goto peek
	}
	if err != nil {
		return nil, err
	}
	goto peek
}

func (b *RingBuffer) Data() []byte {
	if b.r+b.unReadSize <= len(b.buf) {
		return b.buf[b.r : b.r+b.unReadSize]
	}
	return append(b.buf[b.r:], b.buf[:(b.r+b.unReadSize)%len(b.buf)]...)
}

// DataByPos 返回索引值在start和end之间的数据，闭区间
func (b *RingBuffer) DataByPos(start int, end int) []byte {
	// 因为环形缓冲区原因，所以末位置索引值有可能小于开始位置索引
	if end < start {
		return append(b.buf[start:], b.buf[:end+1]...)
	}
	return b.buf[start : end+1]
}

func (b *RingBuffer) PeekBytes(readOffset int, delim byte) ([]byte, error) {
peek:
	for i := 0; i < b.unReadSize-readOffset; i++ {
		if b.buf[(i+b.r+readOffset)%len(b.buf)] == delim {
			return b.DataByPos((b.r+readOffset)%len(b.buf), (i+b.r+readOffset)%len(b.buf)), nil
		}
	}
	if b.unReadSize == len(b.buf) {
		return nil, fmt.Errorf("the unReadSize is over range the buffer len")
	}
	readLen := int(math.Min(float64(b.batchFetchBytes), float64(len(b.buf)-b.unReadSize)))
	buf := make([]byte, readLen)
	readBytes, err := b.reader.Read(buf)
	if readBytes > 0 {
		end := b.r + b.unReadSize + readBytes
		if end < len(b.buf) {
			copy(b.buf[b.r+b.unReadSize:], buf[:readBytes])
		} else {
			writePos := (b.r + b.unReadSize) % len(b.buf)
			n := copy(b.buf[writePos:], buf[:readBytes])
			if n < readBytes {
				copy(b.buf[:end%len(b.buf)], buf[len(b.buf)-writePos:])
			}
		}
		b.unReadSize += readBytes
		goto peek
	}
	if err != nil {
		return nil, err
	}
	goto peek
}

func (b *RingBuffer) SetReadPosition(n int) {
	b.r = (b.r + n) % len(b.buf)
	b.unReadSize -= n
	log.Debug("缓冲区大小 %d rpos=%d %s", b.unReadSize, b.r, string(b.Data()))
}
