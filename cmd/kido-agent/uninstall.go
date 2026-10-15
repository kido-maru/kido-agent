package main

import (
	"fmt"
	"io"
	"os"

	"kido-agent/internal/paths"
	"kido-agent/internal/service"
	"kido-agent/internal/uninstall"
)

func cmdUninstall(out io.Writer) int {
	defer uninstall.PauseIfOwnConsole(out)
	fmt.Fprintln(out, "起動丸エージェントを取り除きます。")
	// 自動起動を先に外す。先に止めると、OS が「落ちた」と見て起こし直すことがある。
	uninstall.RemoveAutostart(out)
	service.StopAll(out)
	if dir, err := paths.ConfigDir(); err == nil {
		// 鍵は秘密なので残さない。ログも一緒に消える。
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintln(out, "! 鍵とログを消せませんでした:", dir, err)
		}
	}
	uninstall.RemoveProgram(out)
	if dir, err := paths.ActionsDir(); err == nil {
		fmt.Fprintf(out, "操作フォルダ %s は残しています。要らなければ手で消してください。\n", dir)
	}
	fmt.Fprintln(out, "✓ 取り除きました")
	return 0
}
