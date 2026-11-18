package wakescreen

import (
	"errors"
	"testing"
	"time"
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

const (
	earlyReason = "smc.sysState.Wake(0x70070000) enet SMC.OutboxNotEmpty"
	magicReason = earlyReason + " Enet.MagicPacket"
)

// newWatcher は時計を手で進められる watcher を作り、動き始めの 1 回を済ませる。
func newWatcher(f *fakeProbe) (*watcher, *time.Time) {
	clock := time.Unix(0, 0)
	w := &watcher{p: f, now: func() time.Time { return clock }}
	w.tick()
	return w, &clock
}

func TestMagicPacketTurnsScreenOn(t *testing.T) {
	f := &fakeProbe{t: 1, reason: magicReason}
	w, _ := newWatcher(f)
	if f.declared != 0 {
		t.Fatal("動き始めたときの起床で点けた")
	}
	f.t = 2
	if !w.tick() || f.declared != 1 {
		t.Fatalf("Magic Packet で起きたのに点けなかった(declared=%d)", f.declared)
	}
	if w.tick() || f.declared != 1 {
		t.Fatalf("同じ起床で 2 度点けた(declared=%d)", f.declared)
	}
}

// 実機では、起きた直後の理由に Enet.MagicPacket がまだ無く、少しあとに書き足された
func TestReasonFilledInLater(t *testing.T) {
	f := &fakeProbe{t: 1, reason: "wifibt"}
	w, clock := newWatcher(f)
	f.t, f.reason = 2, earlyReason
	if w.tick() {
		t.Fatal("理由がそろう前に点けた")
	}
	*clock = clock.Add(2 * time.Second)
	f.reason = magicReason
	if !w.tick() {
		t.Fatal("理由がそろってから点けなかった")
	}
}

func TestGiveUpAfterWindow(t *testing.T) {
	f := &fakeProbe{t: 1, reason: "wifibt"}
	w, clock := newWatcher(f)
	f.t, f.reason = 2, earlyReason
	w.tick()
	*clock = clock.Add(reasonWindow + time.Second)
	f.reason = magicReason // 時間切れのあとに書き足されても、もう点けない
	if w.tick() || f.declared != 0 {
		t.Fatal("時間切れのあとで点けた")
	}
}

func TestOtherReasonsDoNotTurnScreenOn(t *testing.T) {
	for _, reason := range []string{"wifibt", "pwrbtn SMC.OutboxNotEmpty/UserActivity Assertion", "RTC (Maintenance)"} {
		f := &fakeProbe{t: 1, reason: reason}
		w, clock := newWatcher(f)
		f.t = 2
		for i := 0; i < 40; i++ {
			w.tick()
			*clock = clock.Add(500 * time.Millisecond)
		}
		if f.declared != 0 {
			t.Errorf("%q で点けた", reason)
		}
	}
}

func TestDeclareFailureIsNotRetried(t *testing.T) {
	f := &fakeProbe{t: 1, reason: magicReason, err: errors.New("no caffeinate")}
	w, _ := newWatcher(f)
	f.t = 2
	if w.tick() {
		t.Fatal("失敗したのに点けたことにした")
	}
	w.tick()
	if f.declared != 1 {
		t.Fatalf("同じ起床で何度も試した(declared=%d)", f.declared)
	}
}
