package server

import (
	"net/http"
	"time"

	"kido-agent/internal/actions"
	"kido-agent/internal/auth"
)

func (a *agent) handleList(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nonce string `json:"nonce"`
		Sig   string `json:"sig"`
		V     string `json:"v"`
	}
	if readJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "bad_request")
		return
	}
	key := a.authorize(w, req.Nonce, req.Sig, "list", req.Nonce)
	if key == nil {
		return
	}
	list, err := actions.Scan(a.actionsDir, a.goos)
	if err != nil {
		a.logf("操作フォルダを読めません: %v", err)
	}
	body, _ := actions.ListBody(list, a.sessionActive(), req.V)
	writeBody(w, http.StatusOK, body, auth.Sign(key, "list-ok", req.Nonce, string(body)))
}

func (a *agent) handleRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nonce string `json:"nonce"`
		ID    string `json:"id"`
		Sig   string `json:"sig"`
	}
	if readJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "bad_request")
		return
	}
	key := a.authorize(w, req.Nonce, req.Sig, "run", req.Nonce, req.ID)
	if key == nil {
		return
	}
	list, err := actions.Scan(a.actionsDir, a.goos)
	if err != nil {
		a.logf("操作フォルダを読めません: %v", err)
	}
	target, found := actions.FindIn(list, req.ID)
	session, helper := a.sessionActive(), a.helperActive()
	switch {
	case !found:
		writeError(w, http.StatusNotFound, "unknown_action")
		return
	case target.Broken:
		writeError(w, http.StatusConflict, "broken_action")
		return
	case !session && target.RequireLogin:
		// 誰もログインしておらず、ログインしてから使う約束(require_login)の操作。
		writeError(w, http.StatusConflict, "needs_login")
		return
	}
	version := actions.ListVersion(list, session)
	if target.Wait {
		a.runWaited(w, r, key, req.Nonce, version, target, helper)
		return
	}
	body := waitBody(0, nil, version)
	writeBody(w, http.StatusOK, body, auth.Sign(key, "run-ok", req.Nonce, string(body)))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	// スリープやシャットダウンを先に始めると返事が本体に届かないので、
	// 送り切ってから少し待って動かす。手足役がいればそちらへ、いなければ自分で。
	a.logf("実行します: %s (%s, helper=%v)", target.ID, target.Run, helper)
	a.after(a.runDelay, func() { a.execute(target, helper) })
}

// runWaited は wait の操作を動かし、終わってから終了コードで返事をする。
// 常駐アプリ側は時間切れを持たないので、この接続の書き込みの時間切れも外す。
// WriteTimeout は要求を読んだ時点から数えるので、外さないと長く待った後の返事が書けない。
func (a *agent) runWaited(w http.ResponseWriter, r *http.Request, key []byte, nonce, version string, t actions.Action, helper bool) {
	http.NewResponseController(w).SetWriteDeadline(time.Time{})
	a.logf("実行して終わりを待ちます: %s (%s, helper=%v)", t.ID, t.Run, helper)
	// 1 つ空けておき、相手が先に去っても実行の側が送れずに残らないようにする。
	done := make(chan []byte, 1)
	go func() {
		code, err := a.executeWait(r.Context(), t, helper)
		a.logf("終わりました: %s code=%d err=%v", t.ID, code, err)
		done <- waitBody(code, err, version)
	}()
	select {
	case body := <-done:
		writeBody(w, http.StatusOK, body, auth.Sign(key, "run-ok", nonce, string(body)))
	case <-r.Context().Done():
		// 相手が接続を切った。返事は捨てるが、動いているものは止めない。
		a.logf("相手が切ったので結果は返しません(%s は動かしたまま)", t.ID)
	}
}

// waitBody は wait の結果の本文。0 で終われば待たないときと同じ成功の本文にし、
// 本体が wait の有無で読み分けなくて済むようにする。
func waitBody(code int, err error, version string) []byte {
	result := struct {
		OK      bool   `json:"ok"`
		Err     string `json:"err,omitempty"`
		Code    int    `json:"code,omitempty"`
		Version string `json:"v"`
	}{OK: true, Version: version}
	switch {
	case err != nil:
		result.OK, result.Err = false, "start"
	case code != 0:
		result.OK, result.Err, result.Code = false, "exit", code
	}
	return actions.EncodeJSON(result)
}
