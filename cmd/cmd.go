package cmd

import "tinyredis/resp"

type cmdFunc func(args []string) ([]string, byte)

var cmdTable = map[string]cmdFunc{}

func registerCmdFunc(cmd string, f cmdFunc) {
	cmdTable[cmd] = f
}

func RegisterCommands() {
	registerCmdFunc("set", func(args []string) ([]string, byte) {
		return []string{"OK"}, resp.SimpleString
	})
}

func HandleCmd(args []string) ([]string, byte) {
	return cmdTable[args[0]](args)
}
