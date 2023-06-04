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

func (b *RingBuffer) UnReadSize() int {
	return b.unReadSize
}

func (b *RingBuffer) fill() error {
	if b.unReadSize == len(b.buf) {
		return fmt.Errorf("the unReadSize is over range the buffer len")
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
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func (b *RingBuffer) Peek(readOffsetBack, n int) ([]byte, error) {
	if n > len(b.buf) {
		return nil, fmt.Errorf("the unReadSize is over range the buffer len")
	}
peek:
	if n <= b.UnReadSize()-readOffsetBack {
		readPos := (b.r + readOffsetBack) % len(b.buf)
		return b.dataByPos(readPos, (b.r+readOffsetBack+n-1)%len(b.buf)), nil
	}
	err := b.fill()
	if err != nil {
		return nil, err
	}
	goto peek
}

// dataByPos 返回索引值在start和end之间的数据，闭区间
func (b *RingBuffer) dataByPos(start int, end int) []byte {
	// 因为环形缓冲区原因，所以末位置索引值有可能小于开始位置索引
	if end < start {
		return append(b.buf[start:], b.buf[:end+1]...)
	}
	return b.buf[start : end+1]
}

func (b *RingBuffer) PeekBytes(readOffsetBack int, delim byte) ([]byte, error) {
peek:
	for i := 0; i < b.unReadSize-readOffsetBack; i++ {
		if b.buf[(i+b.r+readOffsetBack)%len(b.buf)] == delim {
			return b.dataByPos((b.r+readOffsetBack)%len(b.buf), (i+b.r+readOffsetBack)%len(b.buf)), nil
		}
	}
	err := b.fill()
	if err != nil {
		return nil, err
	}
	goto peek
}

func (b *RingBuffer) SetNextReadPosition(n int) {
	b.r = (b.r + n) % len(b.buf)
	b.unReadSize -= n
}
