package boot

import "io"

// Mac は FileVault が起動時にパスワードを求めるため、ログイン前には動かせない。
// (ディスクの中身が暗号のままで、鍵も操作フォルダも読めない)
const Supported = false

func Enabled() bool             { return false }
func Enable(out io.Writer) int  { return 0 }
func Disable(out io.Writer) int { return 0 }
