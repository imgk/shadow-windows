// +build windows

package main

import (
	"github.com/imgk/shadow-windows/monitor"

	// register protocol
	_ "github.com/imgk/shadow/proto/register"
)

func main() {
	monitor.NewMonitor().Run()
}
