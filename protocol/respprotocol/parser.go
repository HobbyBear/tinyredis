package respprotocol

import (
	"bytes"
	"fmt"
	"strconv"
	"tinyredis/netpoll"
)

const (
	SimpleString = '+'
	Integer      = ':'
	Error        = '-'
	BulkString   = '$'
	Array        = '*'
)

func ReadResp(reader *netpoll.RingBuffer) ([]string, error) {
	cmds := make([]string, 0)
	peekBytes := 0
	readAlign := 0
	line, err := reader.PeekBytes(readAlign, '\n')
	if err != nil {
		return nil, err
	}
	readAlign += len(line)
	length := len(line)
	if length <= 2 || line[length-2] != '\r' {
		return nil, fmt.Errorf("empty line")
	}
	line = bytes.TrimSuffix(line, []byte{'\r', '\n'})
	switch line[0] {
	case SimpleString, Error, Integer:
		cmds = append(cmds, string(line[1:]))
	case BulkString:
		cmds, peekBytes, err = parseBulkString(line, reader, readAlign)
		if err != nil {
			return nil, err
		}
	case Array:
		cmds, peekBytes, err = parseArray(line, reader, readAlign)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("illegle command")
	}
	reader.SetReadPosition(peekBytes + readAlign)
	return cmds, nil
}

var (
	arrayPrefixSlice      = []byte{'*'}
	bulkStringPrefixSlice = []byte{'$'}
	lineEndingSlice       = []byte{'\r', '\n'}
)

func ToBulkStringBytes(arg string) []byte {
	res := bulkStringPrefixSlice
	res = append(res, append([]byte(arg), lineEndingSlice...)...)
	return res
}

func ToSimpleBytes(arg string, t byte) []byte {
	res := []byte{t}
	res = append(res, []byte(arg)...)
	res = append(res, lineEndingSlice...)
	return res
}

func ToArrayBytes(args ...string) []byte {
	res := arrayPrefixSlice
	res = append(res, []byte(strconv.Itoa(len(args)))...)
	res = append(res, lineEndingSlice...)
	// 写入批量字符串
	for _, arg := range args {
		res = append(res, bulkStringPrefixSlice...)
		res = append(res, []byte(strconv.Itoa(len(arg)))...)
		res = append(res, lineEndingSlice...)
		res = append(res, []byte(arg)...)
		res = append(res, lineEndingSlice...)
	}
	return res
}

func parseBulkString(header []byte, reader *netpoll.RingBuffer, readOffset int) ([]string, int, error) {
	cmds := make([]string, 0)
	strLen, err := strconv.ParseInt(string(header[1:]), 10, 64)
	if err != nil || strLen < -1 {
		return cmds, 0, fmt.Errorf("illegal bulk string header: %s", string(header))
	} else if strLen == -1 {
		//payload.Data = MakeNullBulkReply()
		return cmds, 0, fmt.Errorf("unkonw")
	}
	body := make([]byte, strLen+2)
	datalen, err := reader.Peek(readOffset, len(body))
	if err != nil || len(datalen) < len(body) {
		return cmds, len(datalen), err
	}
	cmds = append(cmds, string(body[:len(body)-2]))
	return cmds, len(body), nil
}

func parseArray(header []byte, reader *netpoll.RingBuffer, readOffset int) ([]string, int, error) {
	cmds := make([]string, 0)
	nStrs, err := strconv.ParseInt(string(header[1:]), 10, 64)
	peekbytes := 0
	if err != nil || nStrs < 0 {
		//protocolError(payload, "illegal array header "+string(header[1:]))
		return nil, 0, fmt.Errorf("illegal array header %s", string(header[1:]))
	} else if nStrs == 0 {
		//payload.Data = MakeEmptyMultiBulkReply()
		return nil, 0, fmt.Errorf("nil err")
	}
	lines := make([][]byte, 0, nStrs)
	for i := int64(0); i < nStrs; i++ {
		var line []byte
		line, err = reader.PeekBytes(readOffset, '\n')
		peekbytes += len(line)
		readOffset += len(line)
		if err != nil {
			return nil, peekbytes, err
		}
		length := len(line)
		if length < 4 || line[length-2] != '\r' || line[0] != '$' {
			return nil, peekbytes, fmt.Errorf("illegal bulk string header %s", string(line))
		}
		strLen, err := strconv.ParseInt(string(line[1:length-2]), 10, 64)
		if err != nil || strLen < -1 {
			return nil, peekbytes, fmt.Errorf("illegal bulk string length" + string(line))
		} else if strLen == -1 {
			lines = append(lines, []byte{})
		} else {
			body, err := reader.Peek(readOffset, int(strLen+2))
			peekbytes += len(body)
			readOffset += len(body)
			if err != nil {
				return nil, peekbytes, err
			}
			cmds = append(cmds, string(body[:len(body)-2]))
		}
	}
	return cmds, peekbytes, nil
}
