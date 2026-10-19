//go:build !windows

package desktop

// Interactive は Mac・Linux では常に true。常駐アプリはその利用者の画面の領域
// (LaunchAgent・systemd --user)で動き、セッション 0 にあたる場所が無い。
func Interactive() bool { return true }
