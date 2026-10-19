//go:build !windows

package service

import (
	"testing"

	"kido-agent/internal/control"
)

// Mac・Linux では手足役が生まれないので、ログインしていないと答えていても
// start が手足役を起こしに行ってはいけない(ssh から打ったときなど)。
func TestHelperMissingNeverOffWindows(t *testing.T) {
	if HelperMissing(control.Status{Session: false}) {
		t.Fatal("Windows 以外で手足役がいないと判定した")
	}
}
