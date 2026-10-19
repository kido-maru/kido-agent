package main

import (
	"errors"
	"fmt"
	"io"
	"runtime"

	"kido-agent/internal/control"
)

// firewallHint は「許可」を押し損ねたときに直す場所。
func firewallHint() string {
	switch runtime.GOOS {
	case "windows":
		return "(押し損ねたときは「Windows セキュリティ」→「ファイアウォールとネットワーク保護」→「ファイアウォールによるアプリケーションの許可」で kido-agentd を許可する)"
	case "darwin":
		return "(ファイアウォールを入れている場合は「システム設定」→「ネットワーク」→「ファイアウォール」→「オプション」で kido-agent を許可)"
	}
	return "(ufw を使っている場合は: sudo ufw allow 47821/tcp)"
}

func printPairTimeout(out io.Writer) {
	fmt.Fprintln(out, "✗ 10 分のあいだに本体から届きませんでした。次を確かめてください。")
	fmt.Fprintln(out, "  ・起動ページで、この PC を登録したか")
	fmt.Fprintln(out, "  ・待っている間、起動ページでメニューの「パソコンの操作」を開いていたか")
	fmt.Fprintln(out, "  ・ファイアウォールの確認で「許可」を押したか")
	if hint := firewallHint(); hint != "" {
		fmt.Fprintln(out, "    "+hint)
	}
	fmt.Fprintln(out, "  ・起動丸の本体の電源が入っていて、この PC と同じネットワークにいるか")
	fmt.Fprintln(out, "もう一度組むには、次を実行してください:")
	fmt.Fprintln(out, "  kido-agent setup")
}

// reportControlError は常駐アプリに届かなかったときの案内を出す。
func reportControlError(out io.Writer, err error) {
	if !errors.Is(err, control.ErrNotRunning) {
		fmt.Fprintln(out, "✗", err)
		return
	}
	fmt.Fprintln(out, "✗ 起動丸エージェント(常駐アプリ)が動いていません。")
	fmt.Fprintln(out, "  → kido-agent start で起動できます")
}
