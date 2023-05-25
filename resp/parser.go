package resp

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"runtime/debug"
	"strconv"
	"strings"
	"tinyredis/log"
)

// Payload stores redis.Reply or error
type Payload struct {
	Data Reply
	Err  error
}

// ParseStream reads data from io.Reader and send payloads through channel
func ParseStream(reader *bufio.Reader) <-chan *Payload {
	ch := make(chan *Payload)
	go parse0(reader, ch)
	return ch
}

func parse0(reader *bufio.Reader, ch chan<- *Payload) {
	defer func() {
		if err := recover(); err != nil {
			log.Error(string(debug.Stack()))
		}
	}()
	line, err := reader.ReadBytes('\n')
	if err != nil {
		ch <- &Payload{Err: err}
		close(ch)
		return
	}
	length := len(line)
	if length <= 2 || line[length-2] != '\r' {
		protocolError(ch, "empty line")
		close(ch)
		return
	}
	line = bytes.TrimSuffix(line, []byte{'\r', '\n'})
	switch line[0] {
	case '+':
		content := string(line[1:])
		ch <- &Payload{
			Data: MakeStatusReply(content),
		}
		if strings.HasPrefix(content, "FULLRESYNC") {
			err = parseRDBBulkString(reader, ch)
			if err != nil {
				ch <- &Payload{Err: err}
				close(ch)
				return
			}
		}
	case '-':
		ch <- &Payload{
			Data: MakeErrReply(string(line[1:])),
		}
	case ':':
		value, err := strconv.ParseInt(string(line[1:]), 10, 64)
		if err != nil {
			protocolError(ch, "illegal number "+string(line[1:]))
			return
		}
		ch <- &Payload{
			Data: MakeIntReply(value),
		}
	case '$':
		err = parseBulkString(line, reader, ch)
		if err != nil {
			ch <- &Payload{Err: err}
			close(ch)
			return
		}
	case '*':
		err = parseArray(line, reader, ch)
		if err != nil {
			ch <- &Payload{Err: err}
			close(ch)
			return
		}
	default:
		args := bytes.Split(line, []byte{' '})
		ch <- &Payload{
			Data: MakeMultiBulkReply(args),
		}
	}
}

func parseBulkString(header []byte, reader *bufio.Reader, ch chan<- *Payload) error {
	strLen, err := strconv.ParseInt(string(header[1:]), 10, 64)
	if err != nil || strLen < -1 {
		protocolError(ch, "illegal bulk string header: "+string(header))
		return nil
	} else if strLen == -1 {
		ch <- &Payload{
			Data: MakeNullBulkReply(),
		}
		return nil
	}
	body := make([]byte, strLen+2)
	_, err = io.ReadFull(reader, body)
	if err != nil {
		return err
	}
	ch <- &Payload{
		Data: MakeBulkReply(body[:len(body)-2]),
	}
	return nil
}

// there is no CRLF between RDB and following AOF, therefore it needs to be treated differently
func parseRDBBulkString(reader *bufio.Reader, ch chan<- *Payload) error {
	header, err := reader.ReadBytes('\n')
	header = bytes.TrimSuffix(header, []byte{'\r', '\n'})
	if len(header) == 0 {
		return errors.New("empty header")
	}
	strLen, err := strconv.ParseInt(string(header[1:]), 10, 64)
	if err != nil || strLen <= 0 {
		return errors.New("illegal bulk header: " + string(header))
	}
	body := make([]byte, strLen)
	_, err = io.ReadFull(reader, body)
	if err != nil {
		return err
	}
	ch <- &Payload{
		Data: MakeBulkReply(body[:len(body)]),
	}
	return nil
}

func parseArray(header []byte, reader *bufio.Reader, ch chan<- *Payload) error {
	nStrs, err := strconv.ParseInt(string(header[1:]), 10, 64)
	if err != nil || nStrs < 0 {
		protocolError(ch, "illegal array header "+string(header[1:]))
		return nil
	} else if nStrs == 0 {
		ch <- &Payload{
			Data: MakeEmptyMultiBulkReply(),
		}
		return nil
	}
	lines := make([][]byte, 0, nStrs)
	for i := int64(0); i < nStrs; i++ {
		var line []byte
		line, err = reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		length := len(line)
		if length < 4 || line[length-2] != '\r' || line[0] != '$' {
			protocolError(ch, "illegal bulk string header "+string(line))
			break
		}
		strLen, err := strconv.ParseInt(string(line[1:length-2]), 10, 64)
		if err != nil || strLen < -1 {
			protocolError(ch, "illegal bulk string length "+string(line))
			break
		} else if strLen == -1 {
			lines = append(lines, []byte{})
		} else {
			body := make([]byte, strLen+2)
			_, err := io.ReadFull(reader, body)
			if err != nil {
				return err
			}
			lines = append(lines, body[:len(body)-2])
		}
	}
	ch <- &Payload{
		Data: MakeMultiBulkReply(lines),
	}
	return nil
}

func protocolError(ch chan<- *Payload, msg string) {
	err := errors.New("protocol error: " + msg)
	ch <- &Payload{Err: err}
}
