// +build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/imgk/shadow/app"
	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

const about = `Shadow: A Transparent Proxy for Windows, Linux and macOS
Developed by John Xiong (https://imgk.cc)
https://github.com/imgk/shadow-windows`

type monitor struct {
	// window
	mainWindow *walk.MainWindow
	windowSize walk.Size
	icon       *walk.Icon

	// menus
	menus      []declarative.MenuItem

	// server and mode
	boxServer    *walk.ComboBox
	boxMode      *walk.ComboBox

	// server list and mode list
	servers  []string
	mServers map[string]string
	rules    []string
	mRules   map[string]string

	// button
	buttonStart *walk.PushButton

	// status bar
	barItemStatus *walk.StatusBarItem
	barItems      []declarative.StatusBarItem

	// notify
	notifyIcon *walk.NotifyIcon

	app     *app.App
	mu      sync.Mutex
	running bool
}

func (m *monitor) Run() (err error) {
	m.icon, err = walk.NewIconFromResourceWithSize("$shadow.ico", walk.Size{16, 16})
	if err != nil {
		return
	}

	m.menus = []declarative.MenuItem{
		declarative.Menu{
			Text: "&Server",
			Items: []declarative.MenuItem{
				declarative.Action{
					Text:        "&Manage",
					OnTriggered: nil,
				},
				declarative.Action{
					Text: "E&xit",
					OnTriggered: func() {
						m.stop()
						walk.App().Exit(0)
					},
				},
			},
		},
		declarative.Menu{
			Text: "&Help",
			Items: []declarative.MenuItem{
				declarative.Action{
					Text:        "About",
					OnTriggered: m.about,
				},
			},
		},
	}

	m.barItems = []declarative.StatusBarItem{
		declarative.StatusBarItem{
			AssignTo: &m.barItemStatus,
			Text:     "Shadow is not Running",
		},
	}

	err = declarative.MainWindow{
		AssignTo:       &m.mainWindow,
		Name:           "Shadow",
		Title:          "Shadow: A Transparent Proxy for Windows, Linux and macOS",
		Icon:           m.icon,
		Persistent:     true,
		Layout:         declarative.VBox{},
		MenuItems:      m.menus,
		StatusBarItems: m.barItems,
		Children: []declarative.Widget{
			declarative.GroupBox{
				Title:  "Shadow Config",
				Layout: declarative.VBox{},
				Children: []declarative.Widget{
					declarative.Composite{
						Layout: declarative.Grid{Columns: 2},
						Children: []declarative.Widget{
							declarative.TextLabel{
								Text: "Server",
							},
							declarative.ComboBox{
								AssignTo: &m.boxServer,
								Editable: true,
							},
							declarative.TextLabel{
								Text: "Mode",
							},
							declarative.ComboBox{
								AssignTo: &m.boxMode,
								Editable: true,
							},
						},
					},
				},
			},
			declarative.Composite{
				Layout: declarative.Grid{Columns: 5},
				Children: []declarative.Widget{
					declarative.HSpacer{},
					declarative.PushButton{
						AssignTo: &m.buttonStart,
						Text:     "Start",
						OnClicked: func() {
							if m.running {
								if err := m.stop(); err != nil {
									m.error(err)
								}
								return
							}

							if err := m.start(); err != nil {
								m.error(err)
							}
						},
					},
					declarative.HSpacer{},
					declarative.PushButton{
						Text:     "Generate",
						OnClicked: nil,
					},
					declarative.HSpacer{},
				},
			},
		},
	}.Create()
	if err != nil {
		return
	}

	disableResize := win.GetWindowLong(m.mainWindow.Handle(), win.GWL_STYLE) & ^win.WS_THICKFRAME & ^win.WS_MAXIMIZEBOX
	win.SetWindowLong(m.mainWindow.Handle(), win.GWL_STYLE, disableResize)

	m.mainWindow.SetSize(m.windowSize)
	m.mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		m.mainWindow.Hide()
	})

	m.ReadRules()
	m.ReadServers()

	m.notifyIcon, err = walk.NewNotifyIcon(m.mainWindow)
	if err != nil {
		return
	}
	defer m.notifyIcon.Dispose()

	m.notifyIcon.SetToolTip("A Transparent Proxy for Windows, Linux and macOS")
	m.notifyIcon.SetIcon(m.icon)
	m.notifyIcon.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		switch button {
		case walk.LeftButton:
			m.mainWindow.Show()
		case walk.RightButton:
		case walk.MiddleButton:
		default:
		}
	})

	exitAction := walk.NewAction()
	if er := exitAction.SetText("E&xit"); er != nil {
		err = er
		return
	}
	exitAction.Triggered().Attach(func() {
		m.stop()
		walk.App().Exit(0)
	})
	if er := m.notifyIcon.ContextMenu().Actions().Add(exitAction); er != nil {
		err = er
		return
	}

	m.notifyIcon.SetVisible(true)
	m.mainWindow.Run()
	return nil
}

func (m *monitor) start() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}

	file, err := absFilePath("config.json")
	if err != nil {
		return
	}

	m.app, err = app.NewApp(file, time.Minute, os.Stdout)
	if err != nil {
		return
	}

	fmt.Println("shadow - a transparent proxy for Windows, Linux and macOS")
	fmt.Println("shadow is running...")
	go m.run()

	m.running = true
	m.barItemStatus.SetText("Shadow is Running")
	m.buttonStart.SetText("Stop")
	return
}

func (m *monitor) run() {
	if err := m.app.Run(); err != nil {
		m.error(err)
	}
}

func (m *monitor) stop() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	fmt.Println("shadow is closing...")
	m.app.Close()

	select {
	case <-time.After(time.Second * 10):
		buf := make([]byte, 1024)
		for {
			n := runtime.Stack(buf, true)
			if n < len(buf) {
				buf = buf[:n]
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		lines := bytes.Split(buf, []byte{'\n'})
		fmt.Println("Failed to shutdown after 10 seconds. Probably dead locked. Printing stack and killing.")
		for _, line := range lines {
			if len(bytes.TrimSpace(line)) > 0 {
				fmt.Println(string(line))
			}
		}
		os.Exit(777)
	case <-m.app.Done():
		m.running = false
		m.barItemStatus.SetText("Shadow is not Running")
		m.buttonStart.SetText("Start")
	}
	return
}

func (m *monitor) ReadRules() {
	dir, err := absDirPath("rules")
	if err != nil {
		return
	}
	m.rules, m.mRules, err = readRules(dir)
	if err != nil {
		m.error(err)
		return
	}
	m.boxMode.SetModel(m.rules)
	return
}

func (m *monitor) ReadServers() {
	config, err := absFilePath("servers.json")
	if err != nil {
		return
	}
	b, err := ioutil.ReadFile(config)
	if err != nil {
		m.error(err)
		return
	}

	m.mServers = map[string]string{}
	if err := json.Unmarshal(b, &m.mServers); err != nil {
		m.error(err)
		return
	}
	m.servers = make([]string, 0, len(m.mServers))
	for server, _ := range m.mServers {
		m.servers = append(m.servers, server)
	}
	m.boxServer.SetModel(m.servers)
	return
}

func (m *monitor) about() {
	walk.MsgBox(m.mainWindow, "About", about, walk.MsgBoxIconInformation)
}

func (m *monitor) error(err error) {
	walk.MsgBox(m.mainWindow, "Error", err.Error(), walk.MsgBoxIconError)
}
