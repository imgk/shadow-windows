// +build windows

package monitor

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/imgk/shadow/app"
)

func absFilePath(conf string) (string, error) {
	if filepath.IsAbs(conf) {
		return conf, checkFile(conf)
	}

	dir, err := os.Getwd()
	if err != nil {
		return conf, err
	}
	conf = filepath.Join(dir, conf)

	return conf, checkFile(conf)
}

func checkFile(conf string) error {
	info, err := os.Stat(conf)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return errors.New("not a file")
	}

	return nil
}

func absDirPath(conf string) (string, error) {
	if filepath.IsAbs(conf) {
		return conf, checkDir(conf)
	}

	dir, err := os.Getwd()
	if err != nil {
		return conf, err
	}
	conf = filepath.Join(dir, conf)

	return conf, checkDir(conf)
}

func checkDir(conf string) error {
	info, err := os.Stat(conf)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	return errors.New("not a dir")
}

func readRules(rules string) (s []string, m map[string]string, err error) {
	dirs, err := ioutil.ReadDir(rules)
	if err != nil {
		return
	}

	m = make(map[string]string)
	for _, info := range dirs {
		if info.IsDir() {
			dir := info.Name()

			files, er := ioutil.ReadDir(filepath.Join(rules, dir))
			if er != nil {
				err = er
				return
			}

			for _, info := range files {
				if info.IsDir() {
					continue
				}
				file := info.Name()
				if strings.HasPrefix(file, ".") {
					continue
				}

				file = filepath.Join(dir, file)
				s = append(s, file)
				m[file] = filepath.Join(rules, file)
			}
			continue
		}

		file := info.Name()
		if strings.HasPrefix(file, ".") {
			continue
		}

		s = append(s, file)
		m[file] = filepath.Join(rules, file)
	}
	return
}

func parseRules(file string) (apps, cidr []string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		l, er := r.ReadString('\n')
		if er != nil {
			if errors.Is(er, io.EOF) {
				break
			}
			err = er
			return
		}
		if len(l) == 0 {
			continue
		}

		l = strings.TrimSuffix(strings.TrimSuffix(l, "\n"), "\r")
		if strings.HasPrefix(l, "#") {
			continue
		}
		if strings.HasSuffix(l, ".exe") {
			apps = append(apps, l)
			continue
		}
		if _, _, er := net.ParseCIDR(l); er == nil {
			cidr = append(cidr, l)
		}
	}

	return
}

func generate(server string, apps, cidr []string) (err error) {
	config, err := absFilePath("config.json")
	if err != nil {
		return
	}
	b, err := ioutil.ReadFile(config)
	conf := app.Conf{}
	if err = json.Unmarshal(b, &conf); err != nil {
		return
	}

	type Proto struct {
		Proto string `json:"protocol"`
		URL   string `json:"url"`
	}
	proto, err := func(s string) (Proto, error) {
		u, err := url.Parse(s)
		if err != nil {
			return Proto{}, err
		}
		return Proto{Proto: u.Scheme, URL: s}, nil
	}(server)
	if err != nil {
		return
	}
	bb, err := json.Marshal(&proto)
	if err != nil {
		return
	}

	conf.Server = bb
	conf.IPCIDRRules.Proxy = append(cidr, "198.18.0.0/16")
	conf.AppRules.Proxy = apps

	f, err := os.OpenFile(config, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	b, err = json.MarshalIndent(&conf, "", "    ")
	if err != nil {
		return
	}
	if _, err = f.Write(b); err != nil {
		return
	}
	return
}
