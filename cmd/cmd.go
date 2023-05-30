package cmd

import "tinyredis/resp"

type cmdFunc func(args [][]byte) []byte

var cmdTable = map[string]cmdFunc{}

func registerCmdFunc(cmd string, f cmdFunc) {
	cmdTable[cmd] = f
}

func RegisterCommands() {
	registerCmdFunc("set", func(args [][]byte) []byte {
		return resp.MakeOkReply().ToBytes()
	})
}

func HandleCmd(args [][]byte) []byte {
	return cmdTable[string(args[0])](args)
}
