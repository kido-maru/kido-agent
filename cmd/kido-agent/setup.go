package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"kido-agent/internal/auth"
	"kido-agent/internal/boot"
	"kido-agent/internal/control"
)

// cmdSetup は最初に要ることをまとめて行う: 本体と組む・ログイン前にも使うかを選ぶ。
// 何度打ってもよい。組めていれば組み直すか聞き(既定は組み直さない)、ログイン前の
// 設定は初めて組んだときは「はい」、それ以外は今の状態を既定にして聞く。
// 答えを読めない(台本・CI)ときは何も変えずに進む。
func cmdSetup(out io.Writer, in io.Reader) int {
	r := bufio.NewReader(in)
	st, err := control.Call("GET", "/control/status")
	if err != nil {
		reportControlError(out, err)
		return 1
	}
	first := !st.Paired
	if st.Paired {
		fmt.Fprintln(out, "✓ 起動丸の本体とはもう組めています。")
		// 組み直すと古い鍵が消え、組み終わるまで本体から使えなくなるので、既定は「いいえ」。
		if ask(out, r, "組み直しますか?", false, false, os.Getenv("KIDO_AGENT_REPAIR")) {
			if code := cmdPair(out); code != 0 {
				return code
			}
		}
	} else {
		if code := cmdPair(out); code != 0 {
			return code
		}
		// 組めたばかりなら、次にすること(操作を置く)の場所を見せる。
		if openFolder(out) == 0 {
			fmt.Fprintln(out, "ここにフォルダを作ってスクリプトやショートカットを入れると、スマホに操作が増えます。")
		}
	}
	return setupBoot(out, r, first)
}

// setupBoot はログイン前(電源を入れただけ)でも使うかを聞く。電源を入れた直後に
// 使えるのがこの機能の本来の価値なので、初めて組んだときの既定は「はい」。
// 打ち直したときは今の状態を既定にし、Enter だけで切り替わらないようにする。
func setupBoot(out io.Writer, r *bufio.Reader, first bool) int {
	if !boot.Supported {
		return 0
	}
	fmt.Fprintln(out)
	cur := boot.Enabled()
	want := ask(out, r, "ログイン前(電源を入れただけ)でも使いますか?", cur || first, cur, os.Getenv("KIDO_AGENT_BOOT"))
	switch {
	case want == cur && cur:
		fmt.Fprintln(out, "ログイン前でも使えます(そのまま)。")
	case want == cur:
		fmt.Fprintln(out, "ログインしているときだけ使います(そのまま)。")
	case want:
		return boot.Enable(out)
	default:
		return boot.Disable(out)
	}
	return 0
}

// ask は はい/いいえ を聞く。Enter だけなら def。env が on/yes/off/no なら聞かずに
// それに従い、skip なら聞かずに keep(今のまま)を返す。台本から呼ばれて入力が
// 無いとき(EOF)も keep にする(人が答えていないのに UAC や linger を動かさない)。
func ask(out io.Writer, r *bufio.Reader, q string, def, keep bool, env string) bool {
	switch strings.ToLower(env) {
	case "on", "yes", "y":
		return true
	case "off", "no", "n":
		return false
	case "skip":
		return keep
	}
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	fmt.Fprintf(out, "%s %s ", q, hint)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(out)
		return keep
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes", "はい":
		return true
	case "n", "no", "いいえ":
		return false
	}
	return def
}

// cmdPair は組める時間を開いて、本体が見つけに来るまで待つ。
func cmdPair(out io.Writer) int {
	st, err := control.Call("POST", "/control/pair")
	if err != nil {
		reportControlError(out, err)
		return 1
	}
	// 多くの人は起動ページに PC を登録済みなので、ここで「登録して」とは言わない。
	// 待っていることだけを伝え、見つからずに終わったときに原因の候補を出す。
	fmt.Fprintln(out, "起動丸の本体と組みます(10 分以内)。")
	return waitPairing(out, st, func() (control.Status, error) {
		return control.Call("GET", "/control/status")
	}, time.Sleep)
}

// waitPairing は組めるか時間切れになるまで様子を見る。
// 問い合わせと待ち方を引数にしているのは、テストで 10 分待たずに済ませるため。
func waitPairing(out io.Writer, st control.Status, get func() (control.Status, error), sleep func(time.Duration)) int {
	for {
		switch {
		case st.Paired:
			// 待っている行(全角が多く幅を取る)を消し切るよう、空白で埋めて上書きする
			fmt.Fprintf(out, "\r%-64s\n", "✓ 本体とつながりました")
			return 0
		case st.State != auth.StatePairing:
			fmt.Fprintln(out)
			printPairTimeout(out)
			return 1
		}
		fmt.Fprintf(out, "\r  起動丸の本体が見つけに来るのを待っています… 残り %d:%02d ", st.Remaining/60, st.Remaining%60)
		sleep(time.Second)
		var err error
		if st, err = get(); err != nil {
			fmt.Fprintln(out)
			reportControlError(out, err)
			return 1
		}
	}
}
