package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"kido-agent/internal/auth"
	"kido-agent/internal/control"
)

func TestWaitPairing(t *testing.T) {
	steps := []control.Status{
		{State: auth.StatePairing, Remaining: 599},
		{State: auth.StatePaired, Paired: true},
	}
	var out bytes.Buffer
	get := func() (control.Status, error) { s := steps[0]; steps = steps[1:]; return s, nil }
	code := waitPairing(&out, control.Status{State: auth.StatePairing, Remaining: 600}, get, func(time.Duration) {})
	if code != 0 || !strings.Contains(out.String(), "残り 10:00") || !strings.Contains(out.String(), "✓ 本体とつながりました") {
		t.Fatalf("%d\n%s", code, out.String())
	}
}

func TestWaitPairingTimeout(t *testing.T) {
	var out bytes.Buffer
	get := func() (control.Status, error) { return control.Status{State: auth.StateUnpaired}, nil }
	code := waitPairing(&out, control.Status{State: auth.StatePairing, Remaining: 1}, get, func(time.Duration) {})
	s := out.String()
	if code != 1 || !strings.Contains(s, "登録したか") || !strings.Contains(s, "「許可」") || !strings.Contains(s, "電源") || !strings.Contains(s, "kido-agent setup") {
		t.Fatal(s)
	}
}

func TestAsk(t *testing.T) {
	cases := []struct {
		in, env string
		def     bool
		want    bool
		prompt  bool
	}{
		{"y\n", "", false, true, true},
		{"n\n", "", true, false, true},
		{"\n", "", true, true, true},      // 空は既定
		{"", "", false, false, true},      // 入力が無い(台本から)なら既定
		{"", "", true, true, true},        //   (ログイン前の設定は今の状態が既定)
		{"n\n", "on", false, true, false}, // 環境変数があれば聞かない
		{"y\n", "no", true, false, false},
		{"", "skip", true, true, false},
	}
	for _, c := range cases {
		var out bytes.Buffer
		got := ask(&out, bufio.NewReader(strings.NewReader(c.in)), "Q?", c.def, c.env)
		if got != c.want || strings.Contains(out.String(), "Q?") != c.prompt {
			t.Errorf("in=%q env=%q def=%v: got %v, out=%q", c.in, c.env, c.def, got, out.String())
		}
	}
}
