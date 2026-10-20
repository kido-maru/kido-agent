package server

import (
	"fmt"
	"strings"
	"testing"

	"kido-agent/internal/auth"
)

// nonce が生きていないときは、署名違いと別の印で返す。hello の連打で押し出された相手が
// 「組んでいない」と取り違えず、hello からやり直せるようにするため。
func TestStaleNonce(t *testing.T) {
	ta := pairedAgent(t)
	ta.setLoggedIn(true)
	mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1")
	h := ta.apiHandler()

	_, first := ta.hello(t)
	for i := 0; i < auth.NonceMax; i++ {
		ta.hello(t)
	}
	if r := call(t, h, "POST", "/v1/list", listReq(first, auth.Sign(testKey, "list", first))); r.status != 401 || !strings.Contains(r.body, `"stale_nonce"`) {
		t.Fatalf("押し出された nonce(list): %+v", r)
	}
	if r := call(t, h, "POST", "/v1/run", runReq(first, "10_sleep")); r.status != 401 || !strings.Contains(r.body, `"stale_nonce"`) {
		t.Fatalf("押し出された nonce(run): %+v", r)
	}

	_, n := ta.hello(t)
	ta.advance(auth.NonceTTL + 1)
	if r := call(t, h, "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n))); r.status != 401 || !strings.Contains(r.body, `"stale_nonce"`) {
		t.Fatalf("失効した nonce: %+v", r)
	}
}

func TestBadSigWithFreshNonce(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1")
	h := ta.apiHandler()

	_, n := ta.hello(t)
	if r := call(t, h, "POST", "/v1/list", listReq(n, auth.Sign(testKey, "run", n))); r.status != 401 || !strings.Contains(r.body, `"unauthorized"`) {
		t.Fatalf("署名違い(list): %+v", r)
	}
	// 署名違いでも nonce は使い切るので、2 度目は stale_nonce になる
	if r := call(t, h, "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n))); r.status != 401 || !strings.Contains(r.body, `"stale_nonce"`) {
		t.Fatalf("使い切った nonce: %+v", r)
	}

	_, n = ta.hello(t)
	bad := fmt.Sprintf(`{"nonce":%q,"id":"10_sleep","sig":%q}`, n, auth.Sign(testKey, "run", n, "other"))
	if r := call(t, h, "POST", "/v1/run", bad); r.status != 401 || !strings.Contains(r.body, `"unauthorized"`) {
		t.Fatalf("署名違い(run): %+v", r)
	}
}
