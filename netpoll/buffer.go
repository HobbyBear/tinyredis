package netpoll

import (
	"io"
)

type BufIO struct {
	// todo buf目前会无限增大，后续还是改成ringbuffer
	buf             []byte
	batchFetchBytes int64
	reader          io.Reader
	r               int // 标记读取的位置，buf的元素指针
}

func NewBufIO(size int, rd io.Reader) *BufIO {
	return &BufIO{buf: make([]byte, 0, size), batchFetchBytes: 1024, reader: rd}
}

func (b *BufIO) Peek(readOffset, n int) ([]byte, error) {
peek:
	res := make([]byte, n)
	if n <= len(b.buf)-b.r-readOffset {
		copy(res, b.buf[b.r+readOffset:n+b.r+readOffset])
		return res, nil
	}
	buf := make([]byte, b.batchFetchBytes)
	readBytes, err := b.reader.Read(buf)
	if readBytes > 0 {
		b.buf = append(b.buf, buf[:readBytes]...)
		goto peek
	}
	if err != nil {
		return nil, err
	}
	goto peek
}

func (b *BufIO) PeekBytes(readOffset int, delim byte) ([]byte, error) {
peek:
	for i := b.r + readOffset; i < len(b.buf); i++ {
		if b.buf[i] == delim {
			return b.buf[b.r+readOffset : i+1], nil
		}
	}
	buf := make([]byte, b.batchFetchBytes)
	readBytes, err := b.reader.Read(buf)
	if readBytes > 0 {
		b.buf = append(b.buf, buf[:readBytes]...)
		goto peek
	}
	if err != nil {
		return nil, err
	}
	goto peek
}

func (b *BufIO) SetReadPosition(n int) {
	b.r += n
}
