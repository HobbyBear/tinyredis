package cmd

type cmdFunc func(args [][]byte) [][]byte

var cmdTable = map[string]cmdFunc{}

func registerCmdFunc(cmd string, f cmdFunc) {
	cmdTable[cmd] = f
}

func RegisterCommands() {
	registerCmdFunc("set", func(args [][]byte) [][]byte {
		return nil
	})
}

func HandleCmd(args [][]byte) [][]byte {
	return cmdTable[string(args[0])](args)
}
