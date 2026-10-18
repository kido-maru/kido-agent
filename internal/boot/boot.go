// Package boot は「ログイン前(電源を入れただけ)でも動く」ようにする段取り。
// OS ごとに仕組みが違う(Windows は起動時タスク、Linux は linger、Mac は非対応)。
// 実体は OS ごとのファイル(boot_windows.go など)にあり、setup と status から呼ぶ。
//
//	Supported: この OS で使えるか
//	Enabled:   今ログイン前でも動く設定か
//	Enable / Disable: 切り替える(結果を out に書き、成否を終了コードで返す)
package boot
