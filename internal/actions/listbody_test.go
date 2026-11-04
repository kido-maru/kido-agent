package actions

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestListCountLimit(t *testing.T) {
	var list []Action
	for i := 0; i < 30; i++ {
		list = append(list, Action{ID: fmt.Sprintf("%02d", i), Name: "n"})
	}
	if _, sent := ListBody(list, true, ""); sent != MaxActions {
		t.Fatalf("24 個まで: %d", sent)
	}
	before := ListVersion(list, true)
	list[29].Name = "changed"
	if ListVersion(list, true) == before {
		t.Fatal("数の上限で落ちる操作も版に含める")
	}
	if body, _ := ListBody(nil, true, ""); string(body) != `{"v":"4f53cda1","a":[]}` {
		t.Fatal(string(body))
	}
}

func TestListBodyLoginMarking(t *testing.T) {
	list := []Action{
		{ID: "10_sleep", Name: "s"},
		{ID: "20_lock", Name: "l", RequireLogin: true},
	}
	body, _ := ListBody(list, false, "")
	if got := string(body); got != `{"v":"6201105e","a":[{"id":"10_sleep","n":"s"},{"id":"20_lock","n":"l","l":1}]}` {
		t.Fatalf("未ログイン時:\n%s", got)
	}
	body, _ = ListBody(list, true, "")
	if got := string(body); got != `{"v":"fb18cf70","a":[{"id":"10_sleep","n":"s"},{"id":"20_lock","n":"l"}]}` {
		t.Fatalf("ログイン中:\n%s", got)
	}
}

func TestListBodyCompactKeys(t *testing.T) {
	list := []Action{{ID: "10_sleep", Name: "スリープ<&>"},
		{ID: "40_shutdown", Name: "停止", Icon: "power", Confirm: true, Broken: true, RequireLogin: true}}
	body, _ := ListBody(list, false, "")
	wantArray := `[{"id":"10_sleep","n":"スリープ<&>"},{"id":"40_shutdown","n":"停止","ic":"power","cf":1,"x":1,"l":1}]`
	sum := sha256.Sum256([]byte(wantArray))
	want := fmt.Sprintf(`{"v":"%x","a":%s}`, sum[:4], wantArray)
	if string(body) != want {
		t.Fatalf("got %s\nwant %s", body, want)
	}
}

func TestListVersionChanges(t *testing.T) {
	base := []Action{{ID: "10_sleep", Name: "sleep"}, {ID: "20_lock", Name: "lock"}}
	version := ListVersion(base, true)
	if ListVersion(base, true) != version || ListVersion(base, false) != version {
		t.Fatal("同じ一覧や require_login のないログイン変化では版を変えない")
	}
	for _, field := range []string{"id", "name", "icon", "confirm", "broken", "login", "order", "add"} {
		t.Run(field, func(t *testing.T) {
			list := append([]Action(nil), base...)
			switch field {
			case "id":
				list[0].ID = "11_sleep"
			case "name":
				list[0].Name = "new"
			case "icon":
				list[0].Icon = "moon"
			case "confirm":
				list[0].Confirm = true
			case "broken":
				list[0].Broken = true
			case "login":
				list[0].RequireLogin = true
				if ListVersion(list, true) != version {
					t.Fatal("ログイン中は require_login を版に含めない")
				}
			case "order":
				list[0], list[1] = list[1], list[0]
			case "add":
				list = append(list, Action{ID: "30_new", Name: "new"})
			}
			if ListVersion(list, false) == version {
				t.Fatal("一覧の変化で版が変わらない")
			}
		})
	}
}

func TestListBodyByteBoundary(t *testing.T) {
	// 上限の直前と直後を実際の短い鍵のバイト列で作り、版の材料を独立に確かめる。
	var list []Action
	for i := 0; i < MaxActions; i++ {
		list = append(list, Action{ID: fmt.Sprintf("%02d_%s", i, strings.Repeat("a", 61)),
			Name: strings.Repeat("n", 32), Icon: strings.Repeat("i", 40),
			Confirm: true, Broken: true, RequireLogin: true})
	}
	rawArray := func() string {
		var parts []string
		for _, a := range list {
			parts = append(parts, fmt.Sprintf(`{"id":%q,"n":%q,"ic":%q,"cf":1,"x":1,"l":1}`, a.ID, a.Name, a.Icon))
		}
		return "[" + strings.Join(parts, ",") + "]"
	}
	fullLength := func() int { return len(`{"v":"12345678","a":` + rawArray() + `}`) }
	for i := 0; fullLength() > MaxBodyBytes+1; i++ {
		index := i % len(list)
		list[index].Icon = list[index].Icon[1:]
	}
	if fullLength() != 4097 {
		t.Fatalf("境界の用意: %d", fullLength())
	}
	sum := sha256.Sum256([]byte(rawArray()))
	version := fmt.Sprintf("%x", sum[:4])
	body, sent := ListBody(list, false, "")
	var got struct {
		V string          `json:"v"`
		A json.RawMessage `json:"a"`
	}
	if err := json.Unmarshal(body, &got); err != nil || got.V != version || sent != MaxActions-1 || len(body) > MaxBodyBytes {
		t.Fatalf("切る前の版を返す: sent=%d len=%d body=%s err=%v", sent, len(body), body, err)
	}
	if ListVersion(list[:sent], false) == version {
		t.Fatal("切った後の版を使ってはいけない")
	}
	last := len(list) - 1
	list[last].Name = strings.Repeat("c", 32)
	changed, changedSent := ListBody(list, false, "")
	if ListVersion(list, false) == version || changedSent != sent {
		t.Fatal("本文から落ちる操作の変更も版を変える")
	}
	var changedGot struct {
		A json.RawMessage `json:"a"`
	}
	if err := json.Unmarshal(changed, &changedGot); err != nil || string(changedGot.A) != string(got.A) {
		t.Fatal("見える操作は同じまま版だけ変わるはず")
	}
	list[last].Name = strings.Repeat("n", 32)
	list[last].Icon = list[last].Icon[1:]
	body, sent = ListBody(list, false, "")
	if len(body) != MaxBodyBytes || sent != MaxActions {
		t.Fatalf("4096 バイトは切らない: len=%d sent=%d", len(body), sent)
	}
}

func TestListBodySame(t *testing.T) {
	list := []Action{{ID: "10_sleep", Name: "s"}}
	version := ListVersion(list, true)
	for _, known := range []string{"", "different", strings.ToUpper(version), version} {
		body, sent := ListBody(list, true, known)
		if known == version {
			if string(body) != `{"v":"`+version+`","same":1}` || sent != 0 {
				t.Fatal(string(body), sent)
			}
		} else if !strings.Contains(string(body), `"a":[`) || sent != 1 {
			t.Fatal(string(body), sent)
		}
	}
}

func TestListBodyDuplicateShortNames(t *testing.T) {
	list := []Action{{ID: "10_sleep", Name: "a"}, {ID: "20_sleep", Name: "b"}, {ID: "30_lock", Name: "c"}}
	body, _ := ListBody(list, true, "")
	for _, want := range []string{`{"id":"10_sleep","n":"a","x":1}`, `{"id":"20_sleep","n":"b","x":1}`, `{"id":"30_lock","n":"c"}`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("%s が無い: %s", want, body)
		}
	}
	// 一覧から落ちた操作との重複も数え、不正な ID は数えない。
	list = list[:1]
	for i := 1; i < MaxActions; i++ {
		list = append(list, Action{ID: fmt.Sprintf("%02d_other", i), Name: "n"})
	}
	list = append(list, Action{ID: "99_sleep", Name: "s"})
	body, _ = ListBody(list, true, "")
	if !strings.Contains(string(body), `{"id":"10_sleep","n":"a","x":1}`) || strings.Contains(string(body), "99_sleep") {
		t.Fatal(string(body))
	}
	list[len(list)-1].Invalid = "bad"
	body, _ = ListBody(list, true, "")
	if strings.Contains(string(body), `{"id":"10_sleep","n":"a","x":1}`) {
		t.Fatal("不正な ID を重複に数えた")
	}
}
