package actions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type wireAction struct {
	ID      string `json:"id"`
	Name    string `json:"n"`
	Icon    string `json:"ic,omitempty"`
	Confirm int    `json:"cf,omitempty"`
	Broken  int    `json:"x,omitempty"`
	Login   int    `json:"l,omitempty"`
}

// wireList は一覧と版で同じ省略の決まりを使うための形。重複の判定は切る前に行い、
// 後ろの同名操作が一覧から落ちても、短い名前で押せないことを知らせる。
func wireList(actions []Action, sessionActive bool) []wireAction {
	wire := make([]wireAction, 0, len(actions))
	duplicates := DuplicateShortNames(actions)
	for _, a := range actions {
		if a.Invalid != "" {
			continue
		}
		wire = append(wire, wireAction{ID: a.ID, Name: a.Name, Icon: a.Icon,
			Confirm: mark(a.Confirm), Broken: mark(a.Broken || len(duplicates[ShortName(a.ID)]) > 1),
			Login: mark(!sessionActive && a.RequireLogin)})
	}
	return wire
}

func mark(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ListVersion は切り詰める前の短い鍵の配列から版を作る。list と run で共有し、
// 本文から落ちた操作の変更や、ログイン状態による l の変化も見逃さないようにする。
func ListVersion(actions []Action, sessionActive bool) string {
	return listVersion(wireList(actions, sessionActive))
}

func listVersion(wire []wireAction) string {
	sum := sha256.Sum256(EncodeJSON(wire))
	return hex.EncodeToString(sum[:4])
}

// ListBody は版が同じなら小さな返事だけを返す。違えば、数と本文の上限に
// 収まるまで後ろから落とす。sent は実際に載せた数(same のときは 0)。
func ListBody(actions []Action, sessionActive bool, knownVersion string) (body []byte, sent int) {
	wire := wireList(actions, sessionActive)
	version := listVersion(wire)
	if knownVersion == version {
		return EncodeJSON(struct {
			Version string `json:"v"`
			Same    int    `json:"same"`
		}{version, 1}), 0
	}
	if len(wire) > MaxActions {
		wire = wire[:MaxActions]
	}
	for {
		body = EncodeJSON(struct {
			Version string       `json:"v"`
			Actions []wireAction `json:"a"`
		}{version, wire})
		if len(body) <= MaxBodyBytes || len(wire) == 0 {
			return body, len(wire)
		}
		wire = wire[:len(wire)-1]
	}
}

// EncodeJSON は署名する本文を作る。HTML 向けのエスケープ(< を < にする等)は
// 本体には要らず、バイト数を無駄に増やすだけなので切る。
func EncodeJSON(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err) // 自分で組んだ値しか渡さないので、ここに来たら作りの誤り
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}
