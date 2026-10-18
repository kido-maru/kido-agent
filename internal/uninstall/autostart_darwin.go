package uninstall

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"kido-agent/internal/service"
)

// RemoveAutostart はログイン時の自動起動(LaunchAgent)を外す。
func RemoveAutostart(out io.Writer) {
	plist := service.PlistPath()
	// bootout で止めてから外す。KeepAlive のため、先に止めると起こし直される。
	exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(os.Getuid())+"/"+service.LaunchLabel).Run()
	if err := os.Remove(plist); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(out, "! 自動起動の設定を消せませんでした:", err)
	}
	// 管理者の権限が要るので、ここでは消さずにコマンドを案内する。
	if _, err := os.Stat("/etc/sudoers.d/kido-agent"); err == nil {
		fmt.Fprintln(out, "! 再起動・シャットダウン用の許可が残っています。消すには:")
		fmt.Fprintln(out, "    sudo rm /etc/sudoers.d/kido-agent")
	}
}
