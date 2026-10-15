package service

import "os/exec"

// Unit は install.sh が置く systemd --user のサービスの名前。
const Unit = "kido-agent.service"

func start() error { return exec.Command("systemctl", "--user", "start", Unit).Run() }

// stop は止めるだけで enable は残す。次のログイン(linger なら起動時)にまた動く。
func stop() { exec.Command("systemctl", "--user", "stop", Unit).Run() }
