// +build windows

package main

import "github.com/imgk/shadow-windows/monitor"

func main() {
	monitor.NewMonitor().Run()
}
