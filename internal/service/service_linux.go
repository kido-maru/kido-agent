package service

import "os/exec"

// splitRoles は待ち受け役と手足役が別のプロセスに分かれうるか。
// 常駐アプリは画面の領域で動く 1 つだけで、手足役は生まれない。
const splitRoles = false

// Unit は install.sh が置く systemd --user のサービスの名前。
const Unit = "kido-agent.service"

func start() error { return exec.Command("systemctl", "--user", "start", Unit).Run() }

// stop は止めるだけで enable は残す。次のログイン(linger なら起動時)にまた動く。
func stop() { exec.Command("systemctl", "--user", "stop", Unit).Run() }

// startHere は Windows で画面の分だけを起こす口。手足役は生まれないので呼ばれないが、start と同じにしておく。
func startHere() error { return start() }
