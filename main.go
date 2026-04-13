package main

import "github.com/Reed-yang/node-monitor/cmd"

// Set via ldflags: -X main.version=v1.0.0
var version = "dev"

func main() {
	cmd.Execute(version)
}
