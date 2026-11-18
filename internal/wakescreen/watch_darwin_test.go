package wakescreen

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

// 名前が変わって読めなくなると、黙って画面が点かなくなる。読めることだけ確かめる
// (CI の Mac は眠らないので、中身は問わない)。
func TestDarwinProbeReads(t *testing.T) {
	p := darwinProbe{}
	if _, err := p.wakeTime(); err != nil {
		t.Fatalf("kern.waketime: %v", err)
	}
	// kern.wakereason は CI の Mac には無かった(起動してから眠ったことが無いためとみている)。
	// 実行時は読めなければ見送って次の起床を待つので、ここでは無いことを失敗にしない。
	if _, err := p.wakeReason(); err != nil {
		if errors.Is(err, unix.ENOENT) {
			t.Skipf("kern.wakereason が無い Mac: %v", err)
		}
		t.Fatalf("kern.wakereason: %v", err)
	}
}
