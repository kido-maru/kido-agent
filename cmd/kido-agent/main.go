// kido-agent は起動丸の本体から届く操作を、この PC で実行する常駐アプリ。
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"kido-agent/internal/auth"
	"kido-agent/internal/control"
	"kido-agent/internal/paths"
	"kido-agent/internal/server"
	"kido-agent/internal/service"
)

// version はリリースのビルドで -ldflags -X main.version=... により埋め込む。
var version = "dev"

const usage = `使い方: kido-agent <コマンド>

  setup      起動丸の本体と組み、ログイン前(電源を入れただけ)にも使うかを選ぶ
  status     動いているか・組めているか・操作フォルダ(~/KidoButtons)の中身を確かめる
  start      常駐アプリを起こす
  stop       常駐アプリを止める(次のログインでまた動きます)
  uninstall  このアプリを取り除く(~/KidoButtons は残します)
`

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	cmd := defaultCommand(args[0])
	if len(args) > 1 {
		cmd = args[1]
	}
	switch cmd {
	case "serve": // 自動起動(launchd・systemd・起動時タスク)が使う。人は打たない
		return server.Serve(version)
	case "setup":
		return cmdSetup(os.Stdout, os.Stdin)
	case "status":
		return cmdStatus(os.Stdout)
	case "start":
		return service.Start(os.Stdout)
	case "stop":
		return service.Stop(os.Stdout)
	case "uninstall":
		return cmdUninstall(os.Stdout)
	case "--version": // インストールの台本が版を読む
		fmt.Println("kido-agent", version)
		return 0
	case "":
		cmdSummary(os.Stdout)
		fmt.Print(usage)
		return 0
	case "help", "--help", "-h":
		fmt.Print(usage)
		return 0
	}
	fmt.Fprintf(os.Stderr, "知らないコマンドです: %s\n\n%s", cmd, usage)
	return 2
}

// defaultCommand は引数なしで起動されたときの動きを実行ファイルの名前で決める。
// Windows ではログオン時の自動起動に GUI 版(kido-agentd.exe)を使い、
// 引数を付けずに登録しても常駐するようにしておく。
func defaultCommand(argv0 string) string {
	base := strings.ToLower(filepath.Base(argv0))
	if strings.HasPrefix(base, "kido-agentd") {
		return "serve"
	}
	return ""
}

// cmdSummary は引数なしで打たれたときに、版と今の様子を短く見せる。
func cmdSummary(out io.Writer) {
	st, stErr := control.Call("GET", "/control/status")
	fmt.Fprintf(out, "起動丸エージェント %s\n\n", version)
	checkPairing(out, st, stErr, hasKey())
	fmt.Fprintln(out)
}

func hasKey() bool {
	kp, err := paths.KeyFile()
	if err != nil {
		return false
	}
	key, _ := auth.LoadKey(kp)
	return key != nil
}
