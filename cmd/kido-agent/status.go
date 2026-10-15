package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"runtime"

	"kido-agent/internal/actions"
	"kido-agent/internal/auth"
	"kido-agent/internal/boot"
	"kido-agent/internal/control"
	"kido-agent/internal/paths"
)

// cmdStatus は、動いているか・組めているか・ログイン前の設定・操作フォルダの中身を
// まとめて見せる。困ったときに打つコマンド。✗ が 1 つでもあれば 1 を返す。
func cmdStatus(out io.Writer) int {
	dir, err := paths.ActionsDir()
	if err != nil {
		fmt.Fprintln(out, "✗ ホームフォルダが分かりません:", err)
		return 1
	}
	st, stErr := control.Call("GET", "/control/status")
	fmt.Fprintf(out, "起動丸エージェント %s\n\n", version)
	bad := checkPairing(out, st, stErr, hasKey())
	printBoot(out)
	fmt.Fprintln(out)
	if checkActions(out, dir, runtime.GOOS) {
		bad = true
	}
	if bad {
		return 1
	}
	return 0
}

// checkPairing は動いているかと、組めているかを見せる。常駐アプリが動いていればその答えを、
// 動いていなければ鍵のファイルの有無を手がかりにする。
func checkPairing(out io.Writer, st control.Status, stErr error, hasKey bool) (bad bool) {
	if stErr != nil {
		reportControlError(out, stErr)
		if hasKey {
			fmt.Fprintln(out, "  (鍵は保存されているので、起動すれば本体から使えます)")
		}
		return true
	}
	if st.Version != "" && st.Version != version {
		// 入れ直しても、前の版が動き続けていることがある
		fmt.Fprintf(out, "✓ 動いています(動いているのは %s。新しい版にするには kido-agent stop → start)\n", st.Version)
	} else {
		fmt.Fprintln(out, "✓ 動いています")
	}
	switch st.State {
	case auth.StatePaired:
		fmt.Fprintln(out, "✓ 起動丸の本体と組めています")
	case auth.StatePairing:
		fmt.Fprintf(out, "… 組める時間が開いています(残り %d:%02d)\n", st.Remaining/60, st.Remaining%60)
	default:
		fmt.Fprintln(out, "✗ まだ起動丸の本体と組んでいません")
		fmt.Fprintln(out, "    → kido-agent setup を実行して、スマホの起動ページでこの PC を登録してください")
		return true
	}
	return false
}

// printBoot はログイン前(電源を入れただけ)でも動く設定かを見せる。Mac は非対応なので出さない。
func printBoot(out io.Writer) {
	if !boot.Supported {
		return
	}
	if boot.Enabled() {
		fmt.Fprintln(out, "ログイン前(電源を入れただけ)でも使えます")
	} else {
		fmt.Fprintln(out, "ログインしているときだけ使えます(変えるには kido-agent setup)")
	}
}

// checkActions は操作フォルダを 1 つずつ確かめる。✗ が 1 つでもあれば true を返す。
func checkActions(out io.Writer, dir, goos string) (bad bool) {
	fmt.Fprintln(out, "操作フォルダ:", dir)
	list, err := actions.Scan(dir, goos)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintln(out, "✗ 操作フォルダがありません → kido-agent stop のあと kido-agent start で作り直されます")
		return true
	} else if err != nil {
		fmt.Fprintln(out, "✗ 操作フォルダを読めません:", err)
		return true
	}
	valid := 0
	for _, a := range list {
		switch {
		case a.Invalid != "":
			fmt.Fprintf(out, "✗ %s: 本体に送りません: %s\n    → フォルダ名を変えてください\n", a.ID, a.Invalid)
			bad = true
			continue
		case a.Broken:
			fmt.Fprintf(out, "✗ %s(%s): 押せません\n", a.ID, a.Name)
			bad = true
		case len(a.Warnings) > 0:
			fmt.Fprintf(out, "! %s(%s)→ %s\n", a.ID, a.Name, a.Run)
		default:
			fmt.Fprintf(out, "✓ %s(%s)→ %s\n", a.ID, a.Name, a.Run)
		}
		valid++
		for _, p := range a.Problems {
			fmt.Fprintln(out, "    "+p)
		}
		for _, w := range a.Warnings {
			fmt.Fprintln(out, "    ! "+w)
		}
	}
	if valid == 0 {
		fmt.Fprintln(out, "! 操作がありません。~/KidoButtons にフォルダを作り、ショートカットやスクリプトを置いてください")
	}
	warnSameShortName(out, list)
	if _, sent := actions.ListBody(list, true); sent < valid {
		fmt.Fprintf(out, "! 本体に送れるのは先頭の %d 個までです(%d 個まで・合わせて %d バイトまで)。残りは表示されません\n",
			sent, actions.MaxActions, actions.MaxBodyBytes)
	}
	return bad
}

// warnSameShortName は、番号を除くと同じ名前になる操作を知らせる。ショートカットなどほかのアプリからは
// その名前で操作を呼ぶので、後ろのものは名前では動かせない(先のものが動く)。
func warnSameShortName(out io.Writer, list []actions.Action) {
	first := map[string]string{}
	for _, a := range list {
		if a.Invalid != "" {
			continue
		}
		name := actions.ShortName(a.ID)
		if prev, ok := first[name]; ok {
			fmt.Fprintf(out, "! %s と %s は、番号を除くと同じ名前(%s)です。名前で呼ぶと %s が動きます\n    → どちらかのフォルダ名を変えてください\n",
				prev, a.ID, name, prev)
			continue
		}
		first[name] = a.ID
	}
}
