// +build windows

package monitor

import (
	"golang.org/x/sys/windows"
	"golang.org/x/text/language"
)

func lang() *Language {
	tag := language.English

	if langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME); err == nil {
		tags := make([]language.Tag, 0, len(languages))

		for _, language := range languages {
			tags = append(tags, language.Tag)
		}

		matcher := language.NewMatcher(tags)
		confidence := language.No

		for _, lang := range langs {
			t, i := language.MatchStrings(matcher, lang)
			if _, _, c := matcher.Match(t); c > confidence {
				tag = tags[i]
				confidence = c
			}
		}
	}

	if language, ok := languages[tag.String()]; ok {
		return language
	}

	return languages[language.English.String()]
}

// Language is ...
// support multi language
type Language struct {
	// Tag is ...
	language.Tag

	// AboutInfo is ...
	AboutInfo string
	// TitleInfo is ...
	TitleInfo string

	// MenuServer is ...
	MenuServer string
	// MenuManage is ...
	MenuManage string
	// MenuExit is ...
	MenuExit string

	// MenuHelp is ...
	MenuHelp string
	// MenuAbout is ...
	MenuAbout string

	// StatusOff is ...
	StatusOff string
	// StatusOn is ...
	StatusOn string

	// ConfigPanel is ...
	ConfigPanel string
	// LabelServer is ...
	LabelServer string
	// LabelMode is ...
	LabelMode string

	// ButtonStart is ...
	ButtonStart string
	// ButtonStop is ...
	ButtonStop string
	// ButtonGenerate is ...
	ButtonGenerate string

	// ToolTip is ...
	ToolTip string

	// ActionExit is ...
	ActionExit string

	// AboutTitle is ...
	AboutTitle string
	// Error Title is ...
	ErrorTitle string
}

const aboutEN = `Shadow: A Transparent Proxy for Windows, Linux and macOS
Developed by John Xiong (https://imgk.cc)
https://github.com/imgk/shadow-windows`

const aboutCN = `Shadow: 适用于 Windows, Linux and macOS 的透明代理
由 John Xiong 开发 (https://imgk.cc)
https://github.com/imgk/shadow-windows`

var languages = map[string]*Language{
	language.English.String(): &Language{
		Tag:            language.English,
		AboutInfo:      aboutEN,
		TitleInfo:      "Shadow: A Transparent Proxy for Windows, Linux and macOS",
		MenuServer:     "&Server",
		MenuManage:     "&Manage",
		MenuExit:       "E&xit",
		MenuHelp:       "&Help",
		MenuAbout:      "&About",
		StatusOff:      "Shadow is not Running",
		StatusOn:       "Shadow is Running",
		ConfigPanel:    "Shadow Config",
		LabelServer:    "Server",
		LabelMode:      "Mode",
		ButtonStart:    "Start",
		ButtonStop:     "Stop",
		ButtonGenerate: "Generate",
		ToolTip:        "A Transparent Proxy for Windows, Linux and macOS",
		ActionExit:     "E&xit",
		AboutTitle:     "About",
		ErrorTitle:     "Error",
	},
	language.SimplifiedChinese.String(): &Language{
		Tag:            language.SimplifiedChinese,
		AboutInfo:      aboutCN,
		TitleInfo:      "Shadow: 适用于 Windows, Linux and macOS 的透明代理",
		MenuServer:     "服务器(&S)",
		MenuManage:     "管理服务器(&M)",
		MenuExit:       "退出(&E)",
		MenuHelp:       "帮助(&H)",
		MenuAbout:      "关于(&A)",
		StatusOff:      "Shadow 已停止",
		StatusOn:       "Shadow 正在运行",
		ConfigPanel:    "配置 Shadow",
		LabelServer:    "服务器",
		LabelMode:      "模式",
		ButtonStart:    "开始",
		ButtonStop:     "停止",
		ButtonGenerate: "生成配置",
		ToolTip:        "适用于 Windows, Linux and macOS 的透明代理",
		ActionExit:     "退出(&X)",
		AboutTitle:     "关于",
		ErrorTitle:     "错误",
	},
}
