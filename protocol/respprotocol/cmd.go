package respprotocol

import (
	"strings"
)

type cmdFunc func(args []string) ([]string, byte)

var cmdTable = map[string]cmdFunc{}

func registerCmdFunc(cmd string, f cmdFunc) {
	cmdTable[cmd] = f
}

func RegisterCommands() {
	registerCmdFunc("set", func(args []string) ([]string, byte) {
		// todo
		return []string{"OK"}, SimpleString
	})
}

func handleCmd(args []string) ([]string, byte) {
	exeFunc, ok := cmdTable[strings.ToLower(args[0])]
	if !ok {
		return []string{"Not support cmd " + args[0]}, SimpleString
	}
	return exeFunc(args)
}
