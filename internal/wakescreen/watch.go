// Package wakescreen は、ネットワーク越しに起こされた Mac を画面まで起こす。
//
// Mac は Magic Packet を受けても、画面を点けずに裏で起きるだけ(DarkWake)で、
// 誰も使わなければ 1 分ほどで眠り直す。起こした人からは「起きなかった」と見える。
// DarkWake の間に「利用者が操作中」と宣言すると、画面まで起きる。
package wakescreen

import (
	"log"
	"strings"
	"time"
)

// probe は OS から読むもの・OS にさせることをまとめたもの。テストで差し替える。
type probe interface {
	// wakeTime は最後に起きた時刻。起きるたびに変わる
	wakeTime() (int64, error)
	// wakeReason は最後に起きた理由
	wakeReason() (string, error)
	// declareActive は「利用者が操作中」と宣言して画面を点ける
	declareActive() error
}

// magicPacket は起きた理由に入る印。ほかの理由(定期の目覚め・キーボード・蓋)では
// 点けない。勝手に画面が点くと、夜中に部屋が明るくなる。
const magicPacket = "MagicPacket"

// reasonWindow は、起きてから理由を読み続ける長さ。理由の「Enet.MagicPacket」は
// 起きた直後には入っておらず、少し遅れて書き足される(実機で、起きて 0.1 秒後に
// 読むとまだ無かった)。DarkWake は 1 分ほど続くので、その手前で諦める。
const reasonWindow = 15 * time.Second

// watcher は起きたことに気づき、Magic Packet で起きたときだけ画面を点ける。
type watcher struct {
	p       probe
	now     func() time.Time
	last    int64
	init    bool
	pending time.Time // 理由を読み続けている起床の、気づいた時刻(ゼロなら無し)
}

// tick は 1 回見る。画面を点けたら true。
func (w *watcher) tick() bool {
	if t, err := w.p.wakeTime(); err == nil {
		switch {
		case !w.init:
			// 動き始めたときの起床は、この常駐アプリが見ていない昔のもの。数えない
			w.last, w.init = t, true
		case t != w.last:
			w.last, w.pending = t, w.now()
		}
	}
	if w.pending.IsZero() {
		return false
	}
	if w.now().Sub(w.pending) > reasonWindow {
		w.pending = time.Time{}
		return false
	}
	reason, err := w.p.wakeReason()
	if err != nil || !strings.Contains(reason, magicPacket) {
		return false
	}
	w.pending = time.Time{}
	if err := w.p.declareActive(); err != nil {
		log.Printf("ネットワーク越しに起こされましたが、画面を点けられませんでした: %v", err)
		return false
	}
	log.Printf("ネットワーク越しに起こされたので、画面を点けました")
	return true
}

// loop は止められるまで見続ける。DarkWake は 1 分ほどで終わり、その間に点けたいので
// 細かく見る。読むのは sysctl 2 つだけなので軽い。
func loop(p probe, every time.Duration) {
	w := &watcher{p: p, now: time.Now}
	for {
		w.tick()
		time.Sleep(every)
	}
}
