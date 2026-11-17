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

// watcher は起きたことに気づき、Magic Packet で起きたときだけ画面を点ける。
type watcher struct {
	p    probe
	last int64
	init bool
}

// magicPacket は起きた理由に入る印。ほかの理由(定期の目覚め・キーボード・蓋)では
// 点けない。勝手に画面が点くと、夜中に部屋が明るくなる。
const magicPacket = "MagicPacket"

// tick は 1 回見る。画面を点けたら true。
func (w *watcher) tick() bool {
	t, err := w.p.wakeTime()
	if err != nil {
		return false
	}
	// 動き始めたときの起床は、この常駐アプリが見ていない昔のもの。数えない
	if !w.init {
		w.last, w.init = t, true
		return false
	}
	if t == w.last {
		return false
	}
	reason, err := w.p.wakeReason()
	if err != nil || reason == "" {
		// 理由がまだ読めないときは、次の回にもう一度見る
		return false
	}
	w.last = t
	if !strings.Contains(reason, magicPacket) {
		return false
	}
	if err := w.p.declareActive(); err != nil {
		log.Printf("ネットワーク越しに起こされましたが、画面を点けられませんでした: %v", err)
		return false
	}
	log.Printf("ネットワーク越しに起こされたので、画面を点けました")
	return true
}

// loop は止められるまで見続ける。DarkWake は 1 分ほどで終わるので、1 秒おきに見る。
func loop(p probe, every time.Duration) {
	w := &watcher{p: p}
	for {
		w.tick()
		time.Sleep(every)
	}
}
