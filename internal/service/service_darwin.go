package service

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// LaunchLabel は install.sh が置く LaunchAgent の名前。
const LaunchLabel = "page.kido.agent"

func launchTarget() string { return "gui/" + strconv.Itoa(os.Getuid()) + "/" + LaunchLabel }

// PlistPath は LaunchAgent の設定ファイルの場所。
func PlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", LaunchLabel+".plist")
}

// start は読み込み済みなら起こし直し、stop で外していれば読み込み直す。
func start() error {
	if exec.Command("launchctl", "print", launchTarget()).Run() == nil {
		return exec.Command("launchctl", "kickstart", launchTarget()).Run()
	}
	if _, err := os.Stat(PlistPath()); err != nil {
		return errors.New("自動起動の設定がありません。入れ直してください")
	}
	return exec.Command("launchctl", "bootstrap", "gui/"+strconv.Itoa(os.Getuid()), PlistPath()).Run()
}

// stop は launchd から外す。KeepAlive なので、外さずに止めると起こし直される。
// plist は残るので、次のログインでまた読み込まれる。
func stop() { exec.Command("launchctl", "bootout", launchTarget()).Run() }
