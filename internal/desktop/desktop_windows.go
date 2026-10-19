// Package desktop は、このプロセスがログイン中の画面で動いているかを答える。
// Windows では起動時タスクや SSH から起きたプロセスはセッション 0 にいて、
// そこで動かしたものは画面の前の人に見えない。
package desktop

import "golang.org/x/sys/windows"

// Interactive は、このプロセスがログイン中の画面(セッション 0 以外)にいるか。
func Interactive() bool {
	var id uint32
	if err := windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &id); err != nil {
		return false
	}
	return id != 0
}
