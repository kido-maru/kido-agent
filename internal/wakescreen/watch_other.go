//go:build !darwin

package wakescreen

// Watch は Mac 以外では何もしない。Windows・Linux は Wake on LAN で画面まで起きる。
func Watch() {}
