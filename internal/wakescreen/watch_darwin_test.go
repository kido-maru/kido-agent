package wakescreen

import "testing"

// 名前が変わって読めなくなると、黙って画面が点かなくなる。読めることだけ確かめる
// (CI の Mac は眠らないので、中身は問わない)。
func TestDarwinProbeReads(t *testing.T) {
	p := darwinProbe{}
	if _, err := p.wakeTime(); err != nil {
		t.Fatalf("kern.waketime: %v", err)
	}
	if _, err := p.wakeReason(); err != nil {
		t.Fatalf("kern.wakereason: %v", err)
	}
}
