package resp

import (
	"bytes"
	"errors"
	"runtime/debug"
	"strconv"
	"tinyredis/log"
)

// Payload stores redis.Reply or error
type Payload struct {
	Data Reply
	Err  error
}

// ParseStream reads data from io.Reader and send payloads through channel
func ParseStream(reader *BufIO) *Payload {
	return parse0(reader)
}

func parse0(reader *BufIO) *Payload {
	payload := &Payload{}
	defer func() {
		if err := recover(); err != nil {
			log.Error(string(debug.Stack()))
		}
	}()
	peekBytes := 0
	readAfter := 0
	line, err := reader.PeekBytes(readAfter, '\n')
	if err != nil {
		payload.Err = err
		return payload
	}
	readAfter += len(line)
	length := len(line)
	if length <= 2 || line[length-2] != '\r' {
		protocolError(payload, "empty line")
		return payload
	}
	line = bytes.TrimSuffix(line, []byte{'\r', '\n'})
	switch line[0] {
	case '+':
		content := string(line[1:])
		payload.Data = MakeStatusReply(content)
	case '-':
		payload.Data = MakeErrReply(string(line[1:]))
	case ':':
		value, err := strconv.ParseInt(string(line[1:]), 10, 64)
		if err != nil {
			protocolError(payload, "illegal number "+string(line[1:]))
			return payload
		}
		payload.Data = MakeIntReply(value)
	case '$':
		peekBytes, err = parseBulkString(line, reader, payload, readAfter)
		if err != nil {
			payload.Err = err
			return payload
		}
	case '*':
		peekBytes, err = parseArray(line, reader, payload, readAfter)
		if err != nil {
			payload.Err = err
			return payload
		}
	default:
		protocolError(payload, "illegal command "+string(line[1:]))
	}
	reader.SetReadPosition(peekBytes + readAfter)
	return payload
}

func parseBulkString(header []byte, reader *BufIO, payload *Payload, readOffset int) (int, error) {
	strLen, err := strconv.ParseInt(string(header[1:]), 10, 64)
	if err != nil || strLen < -1 {
		protocolError(payload, "illegal bulk string header: "+string(header))
		return 0, nil
	} else if strLen == -1 {
		payload.Data = MakeNullBulkReply()
		return 0, nil
	}
	body := make([]byte, strLen+2)
	datalen, err := reader.Peek(readOffset, len(body))
	if err != nil || len(datalen) < len(body) {
		return len(datalen), err
	}
	payload.Data = MakeBulkReply(body[:len(body)-2])
	return len(body), nil
}

func parseArray(header []byte, reader *BufIO, payload *Payload, readOffset int) (int, error) {
	nStrs, err := strconv.ParseInt(string(header[1:]), 10, 64)
	peekbytes := 0
	if err != nil || nStrs < 0 {
		protocolError(payload, "illegal array header "+string(header[1:]))
		return 0, nil
	} else if nStrs == 0 {
		payload.Data = MakeEmptyMultiBulkReply()
		return 0, nil
	}
	lines := make([][]byte, 0, nStrs)
	for i := int64(0); i < nStrs; i++ {
		var line []byte
		line, err = reader.PeekBytes(readOffset, '\n')
		peekbytes += len(line)
		readOffset += len(line)
		if err != nil {
			return peekbytes, err
		}
		length := len(line)
		if length < 4 || line[length-2] != '\r' || line[0] != '$' {
			protocolError(payload, "illegal bulk string header "+string(line))
			break
		}
		strLen, err := strconv.ParseInt(string(line[1:length-2]), 10, 64)
		if err != nil || strLen < -1 {
			protocolError(payload, "illegal bulk string length "+string(line))
			break
		} else if strLen == -1 {
			lines = append(lines, []byte{})
		} else {
			body, err := reader.Peek(readOffset, int(strLen+2))
			peekbytes += len(body)
			readOffset += len(body)
			if err != nil {
				return peekbytes, err
			}
			lines = append(lines, body[:len(body)-2])
		}
	}

	payload.Data = MakeMultiBulkReply(lines)
	return peekbytes, nil
}

func protocolError(payload *Payload, msg string) {
	err := errors.New("protocol error: " + msg)
	payload.Err = err
}
