package main

import "runtime"

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
