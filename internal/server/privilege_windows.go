package server

import (
	"errors"
	"log"
	"os"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi32                  = windows.NewLazySystemDLL("advapi32.dll")
	procCreateRestrictedToken = advapi32.NewProc("CreateRestrictedToken")
)

// keptPrivileges は下げたあとも残す権限。UAC が普通のユーザーに渡すものと同じ。
// スリープ・再起動・シャットダウンには SeShutdownPrivilege が要る。
var keptPrivileges = []string{
	"SeShutdownPrivilege",
	"SeChangeNotifyPrivilege",
	"SeUndockPrivilege",
	"SeIncreaseWorkingSetPrivilege",
	"SeTimeZonePrivilege",
}

// 管理者として効く SID。消さずに拒否専用にする(UAC と同じ)。
var adminSIDs = []string{"S-1-5-32-544", "S-1-5-114"}

// dropPrivilege は、昇格(管理者)で起動されていたら、普通のユーザーの権限で
// 自分を同じ引数で起動し直し、親(昇格側)を終わらせる。起動時タスクは管理者で
// 動くが、操作は普通のユーザーの権限で足りる。子は昇格していないので、これは繰り返さない。
// 起動し直したら true(親は終わってよい)。
// elevatedChildEnv は「これは権限を下げて起動し直した子」の目印。二重に下げて
// 起動し直す無限ループを防ぐ。UAC を切った環境では、権限を下げたトークンでも
// IsElevated が真のままになることがあるので、昇格の判定だけには頼らない。
const elevatedChildEnv = "KIDO_AGENT_ELEVATED_CHILD"

func dropPrivilege() bool {
	if os.Getenv(elevatedChildEnv) == "1" {
		log.Printf("権限: 普通のユーザー(起動し直し済み)")
		return false
	}
	token := windows.GetCurrentProcessToken()
	if !token.IsElevated() {
		log.Printf("権限: 普通のユーザー")
		return false
	}
	restricted, err := normalUserToken()
	if err != nil {
		// 落とせなくても動きはするので、管理者のまま続ける
		log.Printf("権限を下げられませんでした(管理者のまま続けます): %v", err)
		return false
	}
	defer restricted.Close()
	if err := relaunchAs(restricted); err != nil {
		log.Printf("権限を下げて起動し直せませんでした(管理者のまま続けます): %v", err)
		return false
	}
	log.Printf("権限: 管理者だったので、普通のユーザーで起動し直しました")
	return true
}

// normalUserToken は、UAC が普通のユーザーに渡すのと同じ形のトークンを作る。
// SAFER(runas /trustlevel)は使わない。SAFER は SeChangeNotifyPrivilege 以外の権限を
// すべて外すので、スリープ・再起動・シャットダウンが権限不足で効かなくなる。
func normalUserToken() (windows.Token, error) {
	// 新しいトークンのアクセス権はこのハンドルと同じになる。あとで持ち主や整合性を
	// 変えるので、TOKEN_ADJUST_DEFAULT も頼む。
	var self windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(),
		windows.TOKEN_DUPLICATE|windows.TOKEN_ASSIGN_PRIMARY|windows.TOKEN_QUERY|windows.TOKEN_ADJUST_DEFAULT, &self)
	if err != nil {
		return 0, err
	}
	defer self.Close()
	token, err := restrictToken(self)
	if err != nil {
		return 0, err
	}
	for _, set := range []func(windows.Token, windows.Token) error{setOwnerAndDacl, setMedium} {
		if err := set(self, token); err != nil {
			token.Close()
			return 0, err
		}
	}
	return token, nil
}

func setMedium(_, token windows.Token) error { return setMediumIntegrity(token) }

// restrictToken は管理者の SID を拒否専用にし、keptPrivileges 以外の権限を消す。
func restrictToken(self windows.Token) (windows.Token, error) {
	sids := make([]windows.SIDAndAttributes, 0, len(adminSIDs))
	for _, s := range adminSIDs {
		sid, err := windows.StringToSid(s)
		if err != nil {
			return 0, err
		}
		sids = append(sids, windows.SIDAndAttributes{Sid: sid})
	}
	remove, err := privilegesToRemove(self)
	if err != nil {
		return 0, err
	}
	var removePtr *windows.LUIDAndAttributes
	if len(remove) > 0 {
		removePtr = &remove[0]
	}
	var token windows.Token
	r, _, e := procCreateRestrictedToken.Call(uintptr(self), 0,
		uintptr(len(sids)), uintptr(unsafe.Pointer(&sids[0])),
		uintptr(len(remove)), uintptr(unsafe.Pointer(removePtr)),
		0, 0, uintptr(unsafe.Pointer(&token)))
	if r == 0 {
		return 0, e
	}
	return token, nil
}

// privilegesToRemove はトークンの権限のうち keptPrivileges に無いものを返す。
func privilegesToRemove(self windows.Token) ([]windows.LUIDAndAttributes, error) {
	kept := make(map[windows.LUID]bool, len(keptPrivileges))
	for _, name := range keptPrivileges {
		var id windows.LUID
		p, err := windows.UTF16PtrFromString(name)
		if err != nil {
			return nil, err
		}
		if err := windows.LookupPrivilegeValue(nil, p, &id); err != nil {
			return nil, err
		}
		kept[id] = true
	}
	var need uint32
	// 大きさを聞くだけの呼び出しは、必ず ERROR_INSUFFICIENT_BUFFER で失敗する
	windows.GetTokenInformation(self, windows.TokenPrivileges, nil, 0, &need)
	if need == 0 {
		return nil, errors.New("トークンの権限の大きさを読めません")
	}
	buf := make([]byte, need)
	if err := windows.GetTokenInformation(self, windows.TokenPrivileges, &buf[0], need, &need); err != nil {
		return nil, err
	}
	var remove []windows.LUIDAndAttributes
	for _, p := range (*windows.Tokenprivileges)(unsafe.Pointer(&buf[0])).AllPrivileges() {
		if !kept[p.Luid] {
			remove = append(remove, windows.LUIDAndAttributes{Luid: p.Luid})
		}
	}
	return remove, nil
}

// setOwnerAndDacl は新しく作るものの持ち主と既定の権限を本人にする。昇格トークンの
// 既定は Administrators で、拒否専用にしたあとは持ち主にできず、自分で作ったものも開けない。
func setOwnerAndDacl(self, token windows.Token) error {
	user, err := self.GetTokenUser()
	if err != nil {
		return err
	}
	owner := struct{ Owner *windows.SID }{user.User.Sid}
	if err := windows.SetTokenInformation(token, windows.TokenOwner,
		(*byte)(unsafe.Pointer(&owner)), uint32(unsafe.Sizeof(owner))); err != nil {
		return err
	}
	system, err := windows.StringToSid("S-1-5-18")
	if err != nil {
		return err
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{
		grantAll(user.User.Sid, windows.TRUSTEE_IS_USER),
		grantAll(system, windows.TRUSTEE_IS_WELL_KNOWN_GROUP),
	}, nil)
	if err != nil {
		return err
	}
	dacl := struct{ DefaultDacl *windows.ACL }{acl}
	return windows.SetTokenInformation(token, windows.TokenDefaultDacl,
		(*byte)(unsafe.Pointer(&dacl)), uint32(unsafe.Sizeof(dacl)))
}

func grantAll(sid *windows.SID, kind windows.TRUSTEE_TYPE) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.GRANT_ACCESS,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  kind,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}

// setMediumIntegrity はトークンの整合性レベルを Medium(S-1-16-8192)にする。
func setMediumIntegrity(token windows.Token) error {
	sid, err := windows.StringToSid("S-1-16-8192")
	if err != nil {
		return err
	}
	label := windows.Tokenmandatorylabel{
		Label: windows.SIDAndAttributes{Sid: sid, Attributes: windows.SE_GROUP_INTEGRITY},
	}
	return windows.SetTokenInformation(token, windows.TokenIntegrityLevel,
		(*byte)(unsafe.Pointer(&label)), label.Size())
}

// relaunchAs は与えたトークンで、自分と同じ実行ファイル・引数の子を起こす。
func relaunchAs(token windows.Token) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// 引数はそのまま渡す。os.Args[0] は実行ファイルの名前に置き換えておく。
	cmdline := windows.ComposeCommandLine(append([]string{exe}, os.Args[1:]...))
	argv, err := windows.UTF16PtrFromString(cmdline)
	if err != nil {
		return err
	}
	// 今の環境に目印を足して渡す。目印があれば、子は二度と起動し直さない。
	block := envBlockWithGuard()
	si := &windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var pi windows.ProcessInformation
	err = windows.CreateProcessAsUser(token, nil, argv, nil, nil, false,
		windows.CREATE_UNICODE_ENVIRONMENT|windows.CREATE_NO_WINDOW|syscall.CREATE_NEW_PROCESS_GROUP,
		block, nil, si, &pi)
	if err != nil {
		return err
	}
	windows.CloseHandle(pi.Thread)
	windows.CloseHandle(pi.Process)
	return nil
}

// envBlockWithGuard は今の環境変数に目印を足し、UTF-16 の環境ブロック
// (二重ヌル終端)にする。目印があれば、起動し直した子は権限降格を繰り返さない。
func envBlockWithGuard() *uint16 {
	env := append(os.Environ(), elevatedChildEnv+"=1")
	var buf []uint16
	for _, e := range env {
		if strings.IndexByte(e, 0) >= 0 {
			continue
		}
		buf = append(buf, utf16.Encode([]rune(e))...)
		buf = append(buf, 0)
	}
	buf = append(buf, 0)
	return &buf[0]
}
