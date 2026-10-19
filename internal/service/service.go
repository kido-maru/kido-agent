// Package service は常駐アプリを起こす・止める。自動起動の仕組み(launchd・
// systemd --user・Windows のログオン時起動)が OS ごとに違うので、実体は
// service_darwin.go などにある。
package service

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"kido-agent/internal/control"
	"kido-agent/internal/desktop"
	"kido-agent/internal/paths"
)

// Start は常駐アプリを起こし、待ち受け役が応えるまで待つ。
func Start(out io.Writer) int {
	if st, err := control.Call("GET", "/control/status"); err == nil {
		if HelperMissing(st) {
			return startHelper(out)
		}
		fmt.Fprintf(out, "もう動いています(%s)\n", st.Version)
		return 0
	}
	if err := start(); err != nil {
		fmt.Fprintln(out, "✗ 起動できませんでした:", err)
		return 1
	}
	if !waitUp(5 * time.Second) {
		fmt.Fprintln(out, "✗ 起動しませんでした。ログ:", logPath())
		return 1
	}
	fmt.Fprintln(out, "✓ 起動しました")
	return 0
}

// HelperMissing は、待ち受け役は動いているのに、この画面で操作を動かす役がいないか。
// Windows で待ち受け役が起動時タスク(画面の無いセッション 0)にいて、手足役が
// 消えているとき(手足役を止めて入れ直したなど)。次のログインまで直らないので、
// start で起こし直せるようにする。
func HelperMissing(st control.Status) bool {
	return splitRoles && !st.Session && desktop.Interactive()
}

// startHelper はこの画面に kido-agentd を起こし、手足役として取りに来るまで待つ。
// start() は起動時タスクがもう動いていれば、この画面の分を起こすだけになる。
func startHelper(out io.Writer) int {
	if err := start(); err != nil {
		fmt.Fprintln(out, "✗ 起動できませんでした:", err)
		return 1
	}
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if st, err := control.Call("GET", "/control/status"); err == nil && st.Session {
			fmt.Fprintln(out, "✓ この画面でも動くようにしました")
			return 0
		}
	}
	fmt.Fprintln(out, "✗ この画面で動く役が起きませんでした。ログ:", logPath())
	return 1
}

// Stop は常駐アプリを止める。自動起動の設定は残すので、次のログイン(か
// kido-agent start)でまた動く。
func Stop(out io.Writer) int {
	_, err := control.Call("GET", "/control/status")
	running := err == nil
	// 先に OS の見張りから外す。窓口から止めるだけだと launchd が起こし直す。
	stop()
	StopAll(out)
	if _, err := control.Call("GET", "/control/status"); err == nil {
		fmt.Fprintln(out, "✗ 止められませんでした。ログ:", logPath())
		return 1
	}
	if !running {
		fmt.Fprintln(out, "もともと動いていませんでした")
		return 0
	}
	fmt.Fprintln(out, "✓ 止めました(また動かすには: kido-agent start)")
	return 0
}

// StopAll は手で打った serve も含めて、窓口から止めてもらい、残った常駐も止める。
// 止まる前に鍵やログを消すと、Windows ではファイルが使用中で消せない(uninstall)。
func StopAll(out io.Writer) {
	if _, err := control.Call("POST", "/control/stop"); err == nil {
		waitDown(3 * time.Second)
	}
	// 待ち受け役が止まっても、手足役が窓口を取り直して残る(Windows)。自分の分を全部止める。
	StopDaemons(out)
}

func waitUp(limit time.Duration) bool {
	for end := time.Now().Add(limit); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if _, err := control.Call("GET", "/control/status"); err == nil {
			return true
		}
	}
	return false
}

func waitDown(limit time.Duration) {
	for end := time.Now().Add(limit); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if _, err := control.Call("GET", "/control/status"); err != nil {
			return
		}
	}
}

func logPath() string {
	dir, _ := paths.ConfigDir()
	return filepath.Join(dir, "kido-agent.log")
}
