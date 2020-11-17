// +build windows

package main

import (
	//protocols
	_ "github.com/imgk/shadow/protocol/http"
	_ "github.com/imgk/shadow/protocol/shadowsocks"
	_ "github.com/imgk/shadow/protocol/socks"
	_ "github.com/imgk/shadow/protocol/trojan"

	"github.com/lxn/walk"
)

func main() {
	(&monitor{
		windowSize:  walk.Size{450, 175},
		running:     false,
	}).Run()
}
