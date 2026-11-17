package wakescreen

import (
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

// Watch は止まらずに見続ける。go で呼ぶ。
func Watch() { loop(darwinProbe{}, 500*time.Millisecond) }

type darwinProbe struct{}

// pmset -g log でも理由は読めるが、読むだけで 4 秒ほどかかり、DarkWake の短い間に
// 間に合わないことがある。sysctl ならすぐ読める。
func (darwinProbe) wakeTime() (int64, error) {
	tv, err := unix.SysctlTimeval("kern.waketime")
	if err != nil {
		return 0, err
	}
	return tv.Sec*1_000_000 + int64(tv.Usec), nil
}

func (darwinProbe) wakeReason() (string, error) { return unix.Sysctl("kern.wakereason") }

// caffeinate -u は「利用者が操作中」の宣言(UserIsActive)を出し、DarkWake から
// 画面まで起こす。-t 1 で 1 秒後に宣言を外すので、その後はいつもどおり眠れる。
func (darwinProbe) declareActive() error {
	return exec.Command("/usr/bin/caffeinate", "-u", "-t", "1").Run()
}
