package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkAction は操作フォルダを作り、中にファイルを置く(中身は名前をそのまま)。
func mkAction(t *testing.T, root, id string, files ...string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		writeFile(t, filepath.Join(dir, f), f)
	}
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scanOne(t *testing.T, root, goos string) Action {
	t.Helper()
	list, err := Scan(root, goos)
	if err != nil || len(list) != 1 {
		t.Fatalf("%v %+v", err, list)
	}
	return list[0]
}

func TestScanOneRunnable(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, "50_Minecraft起動", "start.lnk", "memo.txt", "desktop.ini")
	a := scanOne(t, root, "windows")
	if a.Broken || a.Run != "start.lnk" || a.Name != "Minecraft起動" || a.Icon != "" || a.Confirm {
		t.Fatalf("%+v", a)
	}
}

func TestScanBrokenCases(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, "a_none", "memo.txt")
	mkAction(t, root, "b_two", "a.bat", "b.cmd")
	d := mkAction(t, root, "c_badtoml", "a.bat")
	writeFile(t, d+"/kido.toml", "name = ゲーム\n")
	d = mkAction(t, root, "d_badrun", "a.bat")
	writeFile(t, d+"/kido.toml", "run = \"../x.bat\"\n")
	d = mkAction(t, root, "e_missing", "a.bat")
	writeFile(t, d+"/kido.toml", "run = \"b.bat\"\n")
	d = mkAction(t, root, "f_notrunnable", "a.bat", "memo.txt")
	writeFile(t, d+"/kido.toml", "run = \"memo.txt\"\n")
	list, _ := Scan(root, "windows")
	if len(list) != 6 {
		t.Fatal(len(list))
	}
	for _, a := range list {
		if !a.Broken || len(a.Problems) == 0 {
			t.Errorf("%s は壊れているはず: %+v", a.ID, a)
		}
	}
	if !strings.Contains(list[1].Problems[0], `run = "a.bat"`) {
		t.Errorf("2 つあるときは run の書き方を案内する: %s", list[1].Problems[0])
	}
}

func TestScanTomlRunAndBOM(t *testing.T) {
	root := t.TempDir()
	d := mkAction(t, root, "30_restart", "a.bat", "start.bat")
	toml := "\xEF\xBB\xBFname = \"再起動\"\nicon = \"rotate-cw\"\nconfirm = true\nrun = \"start.bat\"\ncolor = \"red\"\n"
	writeFile(t, d+"/kido.toml", toml)
	a := scanOne(t, root, "windows")
	if a.Broken || a.Run != "start.bat" || a.Name != "再起動" || a.Icon != "rotate-cw" {
		t.Fatalf("%+v", a)
	}
	if !a.Confirm {
		t.Fatal("confirm = true で確認を出す")
	}
	if len(a.Warnings) != 1 || !strings.Contains(a.Warnings[0], "color") {
		t.Fatalf("知らないキーは警告だけ: %v", a.Warnings)
	}
}

func TestScanConfirmTextIsAMistake(t *testing.T) {
	// confirm は true / false だけ。文を書いたら、ほかの書き方の間違いと同じく押せなくする
	root := t.TempDir()
	d := mkAction(t, root, "x", "a.bat")
	writeFile(t, d+"/kido.toml", "confirm = \"本当に実行しますか?\"\n")
	if a := scanOne(t, root, "windows"); !a.Broken || a.Confirm {
		t.Fatalf("%+v", a)
	}
}

func TestScanBadIconDropped(t *testing.T) {
	root := t.TempDir()
	d := mkAction(t, root, "x", "a.bat")
	writeFile(t, d+"/kido.toml", "icon = \"Moon Icon\"\n")
	a := scanOne(t, root, "windows")
	if a.Icon != "" || a.Broken || len(a.Warnings) != 1 {
		t.Fatalf("%+v", a)
	}
}

func TestScanIgnoresDotAndFiles(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, ".hidden", "a.bat")
	mkAction(t, root, "ok", "a.bat", "._a.bat")
	writeFile(t, filepath.Join(root, "loose.bat"), "")
	a := scanOne(t, root, "windows")
	if a.ID != "ok" || a.Broken {
		t.Fatalf("%+v", a)
	}
}

func TestIDRules(t *testing.T) {
	for id, ok := range map[string]bool{
		"10_sleep":              true,
		"マインクラフト":               true,
		strings.Repeat("x", 64): true,
		strings.Repeat("x", 65): false,
		strings.Repeat("あ", 22): false, // 66 バイト
		"a\x01b":                false,
		`a\b`:                   false,
		".x":                    false,
	} {
		if (IDProblem(id) == "") != ok {
			t.Errorf("%q: %s", id, IDProblem(id))
		}
	}
	root := t.TempDir()
	mkAction(t, root, strings.Repeat("x", 70), "a.bat")
	if a := scanOne(t, root, "windows"); a.Invalid == "" {
		t.Fatal("長すぎる ID は送らない")
	}
	if body, _ := ListBody([]Action{{ID: "x", Invalid: "bad"}}, true, ""); string(body) != `{"v":"4f53cda1","a":[]}` {
		t.Fatal(string(body))
	}
}

func TestDefaultName(t *testing.T) {
	for in, want := range map[string]string{"10_sleep": "sleep", "sleep": "sleep", "12_": "12_", "1_2_x": "2_x"} {
		if got := defaultName(in); got != want {
			t.Errorf("%s → %s", in, got)
		}
	}
}

func TestScanMissingDir(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "none"), "linux"); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRequireLoginToml(t *testing.T) {
	root := t.TempDir()
	d := mkAction(t, root, "20_lock", "lock.ps1")
	writeFile(t, d+"/kido.toml", "require_login = true\n")
	a := scanOne(t, root, "windows")
	if !a.RequireLogin || a.Broken || len(a.Warnings) != 0 {
		t.Fatalf("%+v", a)
	}
	// 既定は false(kido.toml が無い)。ログイン前でも動かせる
	root2 := t.TempDir()
	mkAction(t, root2, "60_server", "start.bat")
	if scanOne(t, root2, "windows").RequireLogin {
		t.Fatal("既定は false のはず")
	}
	// 前の版の before_login は知らないキーとして警告だけ
	root3 := t.TempDir()
	d = mkAction(t, root3, "10_sleep", "sleep.ps1")
	writeFile(t, d+"/kido.toml", "before_login = true\n")
	if a := scanOne(t, root3, "windows"); a.Broken || len(a.Warnings) != 1 || !contains(a.Warnings[0], "before_login") {
		t.Fatalf("%+v", a)
	}
}

func TestScreenWarningWithoutRequireLogin(t *testing.T) {
	root := t.TempDir()
	d := mkAction(t, root, "50_game", "game.lnk")
	a := scanOne(t, root, "windows")
	if len(a.Warnings) != 1 || !contains(a.Warnings[0], "見えません") || !contains(a.Warnings[0], "require_login = true") {
		t.Fatalf("%+v", a)
	}
	// require_login を付ければ警告しない
	writeFile(t, d+"/kido.toml", "require_login = true\n")
	if a := scanOne(t, root, "windows"); len(a.Warnings) != 0 {
		t.Fatalf("%+v", a)
	}
	// スクリプトは画面が要らないので警告しない
	root2 := t.TempDir()
	mkAction(t, root2, "60_server", "start.bat")
	if a := scanOne(t, root2, "windows"); len(a.Warnings) != 0 {
		t.Fatalf("%+v", a)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestFindShortNameAndExactID(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, "10_sleep", "sleep.ps1")
	for _, id := range []string{"10_sleep", "sleep"} {
		if a, found := Find(root, "windows", id); !found || a.ID != "10_sleep" || a.Broken {
			t.Fatalf("%s: found=%v %+v", id, found, a)
		}
	}
	mkAction(t, root, "20_sleep", "sleep.ps1")
	a, found := Find(root, "windows", "sleep")
	if !found || a.ID != "10_sleep" || !a.Broken {
		t.Fatalf("同じ短い名前は先の操作を選ぶが断る: found=%v %+v", found, a)
	}
	for _, id := range []string{"10_sleep", "20_sleep"} {
		if a, found := Find(root, "windows", id); !found || a.Broken || a.ID != id {
			t.Fatalf("完全な ID なら動かせる: found=%v %+v", found, a)
		}
	}
	mkAction(t, root, "sleep", "exact.ps1")
	if a, found := Find(root, "windows", "sleep"); !found || a.ID != "sleep" || a.Broken {
		t.Fatalf("完全な ID が短い名前より優先: found=%v %+v", found, a)
	}
	for _, id := range []string{"missing", "../10_sleep", "Sleep"} {
		if a, found := Find(root, "windows", id); found {
			t.Fatalf("存在しない名前: %s %+v", id, a)
		}
	}
}

func TestFindSkipsInvalidAndIgnoresLimits(t *testing.T) {
	list := []Action{{ID: "10_sleep", Invalid: "bad"}, {ID: "20_sleep"}}
	if a, found := FindIn(list, "sleep"); !found || a.ID != "20_sleep" || a.Broken {
		t.Fatalf("不正な ID は重複にも数えない: found=%v %+v", found, a)
	}
	if _, found := FindIn(list, "10_sleep"); found {
		t.Fatal("不正な ID は完全一致でも動かせない")
	}
	root := t.TempDir()
	for i := 0; i < MaxActions; i++ {
		mkAction(t, root, strings.Repeat("a", i+1), "a.ps1")
	}
	mkAction(t, root, "zz_tail", "a.ps1")
	if a, found := Find(root, "windows", "zz_tail"); !found || a.Broken {
		t.Fatalf("一覧から落ちても動かせる: found=%v %+v", found, a)
	}
}
