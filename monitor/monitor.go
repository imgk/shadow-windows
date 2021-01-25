// +build windows

package monitor

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

type monitor struct {
	// window
	mainWindow *walk.MainWindow
	windowSize walk.Size
	icon       *walk.Icon
	lang       *multiLanguage

	// menus
	menus []declarative.MenuItem

	// server and mode
	boxServer *walk.ComboBox
	boxMode   *walk.ComboBox

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

func NewMonitor() *monitor {
	return &monitor{
		windowSize: walk.Size{450, 175},
		running:    false,
	}
}

func (m *monitor) Run() (err error) {
	m.icon, err = walk.NewIconFromResourceWithSize("$shadow.ico", walk.Size{16, 16})
	if err != nil {
		return
	}
	m.lang = lang()

	m.menus = []declarative.MenuItem{
		declarative.Menu{
			Text: m.lang.MenuServer,
			Items: []declarative.MenuItem{
				declarative.Action{
					Text:        m.lang.MenuManage,
					OnTriggered: nil,
				},
				declarative.Action{
					Text: m.lang.MenuExit,
					OnTriggered: func() {
						m.stop()
						walk.App().Exit(0)
					},
				},
			},
		},
		declarative.Menu{
			Text: m.lang.MenuHelp,
			Items: []declarative.MenuItem{
				declarative.Action{
					Text:        m.lang.MenuAbout,
					OnTriggered: m.about,
				},
			},
		},
	}

	m.barItems = []declarative.StatusBarItem{
		declarative.StatusBarItem{
			AssignTo: &m.barItemStatus,
			Text:     m.lang.StatusOff,
		},
	}

	err = declarative.MainWindow{
		AssignTo:       &m.mainWindow,
		Name:           "Shadow",
		Title:          m.lang.TitleInfo,
		Icon:           m.icon,
		Persistent:     true,
		Layout:         declarative.VBox{},
		MenuItems:      m.menus,
		StatusBarItems: m.barItems,
		Children: []declarative.Widget{
			declarative.GroupBox{
				Title:  m.lang.ConfigPanel,
				Layout: declarative.VBox{},
				Children: []declarative.Widget{
					declarative.Composite{
						Layout: declarative.Grid{Columns: 2},
						Children: []declarative.Widget{
							declarative.TextLabel{
								Text: m.lang.LabelServer,
							},
							declarative.ComboBox{
								AssignTo: &m.boxServer,
								Editable: true,
							},
							declarative.TextLabel{
								Text: m.lang.LabelMode,
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
						Text:     m.lang.ButtonStart,
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
						Text:      m.lang.ButtonGenerate,
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

	m.notifyIcon.SetToolTip(m.lang.ToolTip)
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
	if er := exitAction.SetText(m.lang.ActionExit); er != nil {
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
	m.barItemStatus.SetText(m.lang.StatusOn)
	m.buttonStart.SetText(m.lang.ButtonStop)
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
		m.barItemStatus.SetText(m.lang.StatusOff)
		m.buttonStart.SetText(m.lang.ButtonStart)
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
	walk.MsgBox(m.mainWindow, m.lang.AboutTitle, m.lang.AboutInfo, walk.MsgBoxIconInformation)
}

func (m *monitor) error(err error) {
	walk.MsgBox(m.mainWindow, m.lang.ErrorTitle, err.Error(), walk.MsgBoxIconError)
}
