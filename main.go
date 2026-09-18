package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"serve"}
	}
	switch args[0] {
	case "serve":
		fmt.Println("serve mode (TODO)")
	case "run-report":
		fmt.Println("run-report mode (TODO)")
	case "snapshot":
		fmt.Println("snapshot mode (TODO)")
	default:
		fmt.Println("usage: quick-feishu [serve|run-report|snapshot]")
		os.Exit(1)
	}
}
