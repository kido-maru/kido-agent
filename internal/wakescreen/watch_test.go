package wakescreen

import (
	"errors"
	"testing"
)

type fakeProbe struct {
	t        int64
	reason   string
	err      error
	declared int
}

func (f *fakeProbe) wakeTime() (int64, error)    { return f.t, nil }
func (f *fakeProbe) wakeReason() (string, error) { return f.reason, nil }
func (f *fakeProbe) declareActive() error        { f.declared++; return f.err }

const magicReason = "smc.sysState.Wake(0x70070000) enet SMC.OutboxNotEmpty Enet.MagicPacket"

func TestMagicPacketTurnsScreenOn(t *testing.T) {
	f := &fakeProbe{t: 1, reason: magicReason}
	w := &watcher{p: f}
	if w.tick() {
		t.Fatal("動き始めたときの起床で点けた")
	}
	f.t = 2
	if !w.tick() || f.declared != 1 {
		t.Fatalf("Magic Packet で起きたのに点けなかった(declared=%d)", f.declared)
	}
	// 同じ起床では 2 度点けない
	if w.tick() || f.declared != 1 {
		t.Fatalf("同じ起床で 2 度点けた(declared=%d)", f.declared)
	}
}

func TestOtherReasonsDoNotTurnScreenOn(t *testing.T) {
	for _, reason := range []string{"wifibt", "pwrbtn SMC.OutboxNotEmpty/UserActivity Assertion", "RTC (Maintenance)"} {
		f := &fakeProbe{t: 1, reason: reason}
		w := &watcher{p: f}
		w.tick()
		f.t = 2
		if w.tick() || f.declared != 0 {
			t.Errorf("%q で点けた", reason)
		}
	}
}

func TestEmptyReasonIsRetried(t *testing.T) {
	f := &fakeProbe{t: 1}
	w := &watcher{p: f}
	w.tick()
	f.t = 2
	if w.tick() {
		t.Fatal("理由が読めないのに点けた")
	}
	f.reason = magicReason
	if !w.tick() {
		t.Fatal("理由が読めるようになってから点けなかった")
	}
}

func TestDeclareFailureIsNotFatal(t *testing.T) {
	f := &fakeProbe{t: 1, reason: magicReason, err: errors.New("no caffeinate")}
	w := &watcher{p: f}
	w.tick()
	f.t = 2
	if w.tick() {
		t.Fatal("失敗したのに点けたことにした")
	}
	// 同じ起床で何度も試さない
	w.tick()
	if f.declared != 1 {
		t.Fatalf("declared=%d", f.declared)
	}
}
