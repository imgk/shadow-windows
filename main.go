// +build windows

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"

	//protocols
	_ "github.com/imgk/shadow/protocol/http"
	_ "github.com/imgk/shadow/protocol/shadowsocks"
	_ "github.com/imgk/shadow/protocol/socks"
	_ "github.com/imgk/shadow/protocol/trojan"
	_ "github.com/imgk/shadow/windows/res"

	"github.com/imgk/shadow/windows/pkg"
)

const (
	ServiceName = "Shadow"
	ServiceDesc = "A Transparent Proxy for Windows, Linux and macOS"
)

func main() {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Panic(fmt.Errorf("failed to determine if we are running in an interactive session: %v", err))
	}
	if !isIntSess {
		file := flag.String("c", "config.json", "config file")
		flag.Parse()

		elog, err := eventlog.Open(ServiceName)
		if err != nil {
			log.Panic(err)
		}
		defer elog.Close()

		if err := svc.Run(ServiceName, pkg.Service{Log: elog, File: *file}); err != nil {
			elog.Error(1, fmt.Sprintf("%s service failed: %v", ServiceName, err))
			return
		}

		elog.Info(1, fmt.Sprintf("%s service stopped", ServiceName))
		return
	}

	if len(os.Args) < 2 {
		monitor := &pkg.Monitor{
			ServiceName: ServiceName,
			ServiceDesc: ServiceDesc,
		}
		if err := monitor.Run(); err != nil {
			log.Panic(err)
		}
		return
	}

	switch os.Args[1] {
	case "/install":
		conf, err := pkg.GetConfig("config.json")
		if err != nil {
			log.Panic(err)
		}
		if err := pkg.InstallService(ServiceName, ServiceDesc, []string{"-c", conf}); err != nil {
			log.Panic(err)
		}
		fmt.Println("service installed successfully")
	case "/remove":
		if err := pkg.RemoveService(ServiceName); err != nil {
			log.Panic(err)
		}
		fmt.Println("service removed successfully")
	case "/start":
		if err := pkg.StartService(ServiceName); err != nil {
			log.Panic(err)
		}
		fmt.Println("service started successfully")
	case "/stop":
		if err := pkg.ControlService(ServiceName, svc.Stop, svc.Stopped); err != nil {
			log.Panic(err)
		}
		fmt.Println("service stopped successfully")
	default:
		monitor := &pkg.Monitor{
			ServiceName: ServiceName,
			ServiceDesc: ServiceDesc,
		}
		if err := monitor.Run(); err != nil {
			log.Panic(err)
		}
	}
}
