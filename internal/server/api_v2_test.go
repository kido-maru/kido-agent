package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"kido-agent/internal/actions"
	"kido-agent/internal/auth"
)

func responseVersion(t *testing.T, body string) string {
	t.Helper()
	var result struct {
		V string `json:"v"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	decoded, err := hex.DecodeString(result.V)
	if err != nil || len(decoded) != 4 || result.V != strings.ToLower(result.V) {
		t.Fatalf("版は小文字の 8 桁: %s", body)
	}
	return result.V
}

func fetchList(t *testing.T, ta *testAgent, known *string) apiResult {
	t.Helper()
	_, nonce := ta.hello(t)
	req := listReq(nonce, auth.Sign(testKey, "list", nonce))
	if known != nil {
		req = strings.TrimSuffix(req, "}") + fmt.Sprintf(`,"v":%q}`, *known)
	}
	result := call(t, ta.apiHandler(), "POST", "/v1/list", req)
	if result.status != 200 || result.sig != auth.Sign(testKey, "list-ok", nonce, result.body) {
		t.Fatalf("一覧と same の本文を署名する: %+v", result)
	}
	responseVersion(t, result.body)
	return result
}

func TestListSameSigned(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1")
	first := fetchList(t, ta, nil)
	version := responseVersion(t, first.body)
	for _, known := range []string{"", "different", strings.ToUpper(version), version} {
		result := fetchList(t, ta, &known)
		if known == version {
			if result.body != `{"v":"`+version+`","same":1}` {
				t.Fatal(result.body)
			}
		} else if result.body != first.body {
			t.Fatalf("v が無い・違うなら一覧: %+v", result)
		}
	}
	// 要求の版を署名に混ぜた場合は、従来と同じ署名として扱わない。
	_, nonce := ta.hello(t)
	req := fmt.Sprintf(`{"nonce":%q,"sig":%q,"v":%q}`, nonce, auth.Sign(testKey, "list", nonce, version), version)
	if r := call(t, ta.apiHandler(), "POST", "/v1/list", req); r.status != 401 || !strings.Contains(r.body, "unauthorized") {
		t.Fatalf("v は署名の材料に含めない: %+v", r)
	}
}

func TestEmptyListV2(t *testing.T) {
	ta := pairedAgent(t)
	if r := fetchList(t, ta, nil); r.body != `{"v":"4f53cda1","a":[]}` {
		t.Fatal(r.body)
	}
}

func TestListSameBeforeTruncation(t *testing.T) {
	ta := pairedAgent(t)
	for i := 0; i < actions.MaxActions; i++ {
		d := mkAction(t, ta.actionsDir, fmt.Sprintf("%02d_%s", i, strings.Repeat("a", 60)), "run.ps1")
		writeFile(t, d+"/kido.toml", "name = \""+strings.Repeat("名", 32)+"\"\nicon = \""+strings.Repeat("a", 40)+"\"\n")
	}
	first := fetchList(t, ta, nil)
	var before struct {
		A json.RawMessage `json:"a"`
	}
	if err := json.Unmarshal([]byte(first.body), &before); err != nil {
		t.Fatal(err)
	}
	var visible []json.RawMessage
	if err := json.Unmarshal(before.A, &visible); err != nil || len(visible) >= actions.MaxActions || len(first.body) > actions.MaxBodyBytes {
		t.Fatalf("本文を切り詰める: %s", first.body)
	}
	version := responseVersion(t, first.body)
	if same := fetchList(t, ta, &version); same.body != `{"v":"`+version+`","same":1}` {
		t.Fatal(same.body)
	}
	mkAction(t, ta.actionsDir, "99_tail", "tail.ps1")
	changed := fetchList(t, ta, &version)
	var after struct {
		A json.RawMessage `json:"a"`
	}
	if err := json.Unmarshal([]byte(changed.body), &after); err != nil || string(before.A) != string(after.A) || responseVersion(t, changed.body) == version {
		t.Fatalf("落ちる操作を足しても新しい版と一覧を返す: %s", changed.body)
	}
}

func TestListAndRunShareVersion(t *testing.T) {
	for _, loggedIn := range []bool{false, true} {
		for _, kind := range []string{"immediate", "wait-ok", "wait-exit", "wait-start"} {
			t.Run(fmt.Sprintf("%t/%s", loggedIn, kind), func(t *testing.T) {
				ta := pairedAgent(t)
				ta.osSession = func() bool { return loggedIn }
				d := mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1")
				if kind != "immediate" {
					writeFile(t, d+"/kido.toml", "wait = true\n")
				}
				d = mkAction(t, ta.actionsDir, "20_lock", "lock.ps1")
				writeFile(t, d+"/kido.toml", "require_login = true\n")
				ta.launchWait = func(actions.Action) (int, error) {
					if kind == "wait-exit" {
						return 3, nil
					}
					if kind == "wait-start" {
						return 0, errors.New("起動できない")
					}
					return 0, nil
				}
				list := fetchList(t, ta, nil)
				version := responseVersion(t, list.body)
				_, nonce := ta.hello(t)
				r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(nonce, "sleep"))
				if r.status != 200 || responseVersion(t, r.body) != version || r.sig != auth.Sign(testKey, "run-ok", nonce, r.body) {
					t.Fatalf("list と run で版を合わせる: list=%s run=%+v", list.body, r)
				}
				mkAction(t, ta.actionsDir, "30_new", "new.ps1")
				_, nonce = ta.hello(t)
				r = call(t, ta.apiHandler(), "POST", "/v1/run", runReq(nonce, "10_sleep"))
				newVersion := responseVersion(t, r.body)
				if newVersion == version || newVersion != responseVersion(t, fetchList(t, ta, nil).body) {
					t.Fatalf("run でも追加に気づける: %+v", r)
				}
			})
		}
	}
}

func TestRunShortNameAndDuplicates(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1")
	run := func(id string, status int, errorCode, launched string) {
		t.Helper()
		before := len(ta.launched)
		_, nonce := ta.hello(t)
		r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(nonce, id))
		if r.status != status || (errorCode != "" && r.body != `{"ok":false,"error":"`+errorCode+`"}`) {
			t.Fatalf("%s: %+v", id, r)
		}
		if launched == "" {
			if len(ta.launched) != before {
				t.Fatal("断った操作が動いた")
			}
		} else if len(ta.launched) != before+1 || ta.launched[before] != launched {
			t.Fatalf("%s: %v", id, ta.launched)
		}
	}
	run("sleep", 200, "", "10_sleep/sleep.ps1")
	run("missing", 404, "unknown_action", "")
	mkAction(t, ta.actionsDir, "20_sleep", "second.ps1")
	list := fetchList(t, ta, nil)
	for _, want := range []string{`{"id":"10_sleep","n":"sleep","x":1}`, `{"id":"20_sleep","n":"sleep","x":1}`} {
		if !strings.Contains(list.body, want) {
			t.Fatal(list.body)
		}
	}
	run("sleep", 409, "broken_action", "")
	run("10_sleep", 200, "", "10_sleep/sleep.ps1")
	run("20_sleep", 200, "", "20_sleep/second.ps1")
	mkAction(t, ta.actionsDir, "sleep", "exact.ps1")
	run("sleep", 200, "", "sleep/exact.ps1")
}
