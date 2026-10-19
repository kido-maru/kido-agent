package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// splitRoles は待ち受け役と手足役が別のプロセスに分かれうるか。
// 起動時タスク(セッション 0)の待ち受け役と、画面ごとの手足役に分かれる。
const splitRoles = true

// start は、ログイン前対応の起動時タスクがあればそれを先に起こして待ち受け役に
// させ、この画面用に kido-agentd を起こす(タスクが無ければこちらが待ち受け役になる)。
func start() error {
	if hiddenCmd("schtasks", "/query", "/tn", "KidoAgent").Run() == nil {
		hiddenCmd("schtasks", "/run", "/tn", "KidoAgent").Run()
		waitUp(3 * time.Second)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := hiddenCmd(filepath.Join(filepath.Dir(exe), "kido-agentd.exe"))
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// stop は何もしない。Windows には起こし直す見張りが無く、StopAll で足りる。
func stop() {}

func hiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	return cmd
}
