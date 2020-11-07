// +build windows

package pkg

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	"golang.org/x/sys/windows/svc"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

const about = `Shadow: A Transparent Proxy for Windows, Linux and macOS
Developed by John Xiong (https://imgk.cc)
https://github.com/imgk/shadow
`

type Monitor struct {
	ServiceName string
	ServiceDesc string

	mainWindow     *walk.MainWindow
	boxServer      *walk.ComboBox
	boxMode        *walk.ComboBox
	buttonInstall  *walk.PushButton
	buttonRemove   *walk.PushButton
	buttonGenerate *walk.PushButton
	buttonStart    *walk.PushButton
	buttonStop     *walk.PushButton
	itemStatus     *walk.StatusBarItem
	windowSize     walk.Size
	menus          []declarative.MenuItem
	statusBars     []declarative.StatusBarItem
	servers        []string
	mServers       map[string]string
	rules          []string
	mRules         map[string]string

	notifyIcon *walk.NotifyIcon
	icon       *walk.Icon
}

func (m *Monitor) Run() (err error) {
	if m.icon, err = walk.NewIconFromResourceWithSize("$shadow.ico", walk.Size{16, 16}); err != nil {
		return
	}

	m.windowSize = walk.Size{550, 175}
	m.menus = []declarative.MenuItem{
		declarative.Menu{
			Text: "&Server",
			Items: []declarative.MenuItem{
				declarative.Action{
					Text:        "&Server",
					OnTriggered: nil,
				},
				declarative.Action{
					Text:        "E&xit",
					OnTriggered: m.exit,
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

	m.statusBars = []declarative.StatusBarItem{
		declarative.StatusBarItem{
			AssignTo: &m.itemStatus,
			Text:     "",
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
		StatusBarItems: m.statusBars,
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
					declarative.PushButton{
						AssignTo:  &m.buttonStart,
						Text:      "Install",
						OnClicked: m.install,
					},
					declarative.PushButton{
						AssignTo:  &m.buttonStop,
						Text:      "Remove",
						OnClicked: m.remove,
					},
					declarative.PushButton{
						AssignTo:  &m.buttonGenerate,
						Text:      "Generate",
						OnClicked: m.generate,
					},
					declarative.PushButton{
						AssignTo:  &m.buttonStart,
						Text:      "Start",
						OnClicked: m.start,
					},
					declarative.PushButton{
						AssignTo:  &m.buttonStop,
						Text:      "Stop",
						OnClicked: m.stop,
					},
				},
			},
		},
	}.Create()
	if err != nil {
		return
	}

	newStyle := win.GetWindowLong(m.mainWindow.Handle(), win.GWL_STYLE) & ^win.WS_THICKFRAME & ^win.WS_MAXIMIZEBOX
	win.SetWindowLong(m.mainWindow.Handle(), win.GWL_STYLE, newStyle)

	m.mainWindow.SetSize(m.windowSize)
	m.mainWindow.Closing().Attach(m.close)
	m.loadRules()
	m.loadServers()
	m.setStatus()

	if m.notifyIcon, err = walk.NewNotifyIcon(m.mainWindow); err != nil {
		return
	}
	defer m.notifyIcon.Dispose()

	m.notifyIcon.SetToolTip("A Transparent Proxy for Windows, Linux and macOS")
	m.notifyIcon.SetIcon(m.icon)
	m.notifyIcon.MouseDown().Attach(m.mouseClicked)

	exitAction := walk.NewAction()
	if err = exitAction.SetText("E&xit"); err != nil {
		return
	}
	exitAction.Triggered().Attach(m.exit)
	if err = m.notifyIcon.ContextMenu().Actions().Add(exitAction); err != nil {
		return
	}

	m.notifyIcon.SetVisible(true)
	m.mainWindow.Run()
	return nil
}

func (m *Monitor) mouseClicked(x, y int, button walk.MouseButton) {
	if button == walk.LeftButton {
		m.mainWindow.Show()
	}
}

func (m *Monitor) exit() {
	ControlService(m.ServiceName, svc.Stop, svc.Stopped)
	walk.App().Exit(0)
}

func (m *Monitor) close(canceled *bool, reason walk.CloseReason) {
	*canceled = true
	m.mainWindow.Hide()
}

func (m *Monitor) show() {
	m.mainWindow.SetSize(m.windowSize)
	m.setStatus()
	m.mainWindow.Show()
}

func (m *Monitor) install() {
	conf, err := GetConfig("config.json")
	if err != nil {
		m.error(err)
		return
	}
	exist, err := IsExistService(m.ServiceName)
	if err != nil {
		m.error(err)
		return
	}
	if exist {
		m.info("Service is Installed...")
		return
	}
	if err := InstallService(m.ServiceName, m.ServiceDesc, []string{"-c", conf}); err != nil {
		m.error(err)
		return
	}
	m.info("Service is Installed...")
}

func (m *Monitor) remove() {
	exist, err := IsExistService(m.ServiceName)
	if err != nil {
		m.error(err)
		return
	}
	if !exist {
		m.info("Service is Removed...")
		return
	}
	if err := RemoveService(m.ServiceName); err != nil {
		m.error(err)
		return
	}
	m.info("Service is Removed...")
}

func (m *Monitor) generate() {
	server := m.boxServer.Text()
	if server == "" {
		m.error(errors.New("Please select a server"))
		return
	}
	mode := m.boxMode.Text()
	if mode == "" {
		m.error(errors.New("Please select a mode"))
		return
	}
	apps, cidr, err := ParseRules(m.mRules[mode])
	if err != nil {
		m.error(err)
		return
	}
	if err := Generate(m.mServers[server], apps, cidr); err != nil {
		m.error(err)
		return
	}
	m.info("Config is Generated...")
}

func (m *Monitor) start() {
	running, err := IsRunningService(m.ServiceName)
	if err != nil {
		m.error(err)
		return
	}
	if running {
		m.info("Service is Running...")
		return
	}
	if err := StartService(m.ServiceName); err != nil {
		m.error(err)
		return
	}
	m.info("Service is Running...")
	time.Sleep(500 * time.Millisecond)
	m.setStatus()
}

func (m *Monitor) stop() {
	running, err := IsRunningService(m.ServiceName)
	if err != nil {
		m.error(err)
		return
	}
	if !running {
		m.info("Service is Stopped...")
		return
	}
	if err := ControlService(m.ServiceName, svc.Stop, svc.Stopped); err != nil {
		m.error(err)
		return
	}
	m.info("Service is Stopped...")
	m.setStatus()
}

func (m *Monitor) loadRules() {
	rules, err := GetRuleDir("rules")
	if err != nil {
		return
	}
	m.rules, m.mRules, err = GetRules(rules)
	if err != nil {
		m.error(err)
		return
	}
	m.boxMode.SetModel(m.rules)
	return
}

func (m *Monitor) loadServers() {
	config, err := GetConfig("servers.json")
	if err != nil {
		return
	}
	b, err := ioutil.ReadFile(config)
	if err != nil {
		m.error(err)
		return
	}

	servers := map[string]string{}
	if err := json.Unmarshal(b, &servers); err != nil {
		m.error(err)
		return
	}
	keys := make([]string, 0, len(servers))
	for k, _ := range servers {
		keys = append(keys, k)
	}
	m.servers = keys
	m.boxServer.SetModel(keys)
	return
}

func (m *Monitor) setStatus() {
	running, err := IsRunningService(m.ServiceName)
	if err != nil {
		m.error(err)
		return
	}
	if running {
		m.itemStatus.SetText("Shadow is Running")
		return
	}
	m.itemStatus.SetText("Shadow is not Running")
}

func (m *Monitor) about() {
	walk.MsgBox(m.mainWindow, "About", about, walk.MsgBoxIconInformation)
}

func (m *Monitor) info(msg string) {
	walk.MsgBox(m.mainWindow, "Info", msg, walk.MsgBoxIconInformation)
}

func (m *Monitor) error(err error) {
	walk.MsgBox(m.mainWindow, "Error", err.Error(), walk.MsgBoxIconError)
}
