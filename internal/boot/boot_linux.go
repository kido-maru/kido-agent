package boot

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Linux は linger(ログインしていなくても user のサービスを動かし続ける設定)で、
// ログイン前でも待ち受け役を動かせる。root 権限が要るときは案内する。
const Supported = true

func Enable(out io.Writer) int  { return setLinger(out, true) }
func Disable(out io.Writer) int { return setLinger(out, false) }

func setLinger(out io.Writer, on bool) int {
	verb := "enable-linger"
	if !on {
		verb = "disable-linger"
	}
	user := currentUser()
	if err := exec.Command("loginctl", verb, user).Run(); err != nil {
		// 普通は本人のぶんは権限なしでできるが、環境によっては root が要る
		fmt.Fprintf(out, "✗ 設定できませんでした。次を試してください:\n    sudo loginctl %s %s\n", verb, user)
		return 1
	}
	if on {
		fmt.Fprintln(out, "✓ ログイン前でも動くようにしました(linger)")
	} else {
		fmt.Fprintln(out, "✓ ログイン前には動かさないようにしました")
	}
	return 0
}

func Enabled() bool {
	cmd := exec.Command("loginctl", "show-user", currentUser(), "-p", "Linger", "--value")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()
	return strings.TrimSpace(buf.String()) == "yes"
}

func currentUser() string {
	out, err := exec.Command("id", "-un").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
