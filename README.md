# 起動丸エージェント(kido-agent)

起動丸(Kido)で、スマホからこの PC をスリープ・画面ロック・再起動・シャットダウンできるようにする常駐アプリです。
ゲームの起動など、自分で作った操作も足せます。
Windows・Mac・Linux に対応しています(Linux は Ubuntu で動作確認)。

使いかた(操作の足しかた・ログイン前でも使う・困ったとき)は、起動丸の説明書にあります: https://docs.kido.page/use/pc-actions/

## インストール

管理者の権限は要りません。入れ直すと更新になります。

Windows(PowerShell に貼り付け):

```powershell
irm https://github.com/kido-maru/kido-agent/releases/latest/download/install.ps1 | iex
```

Mac・Linux(ターミナルに貼り付け):

```sh
curl -fsSL https://github.com/kido-maru/kido-agent/releases/latest/download/install.sh | sh
```

版を決めて入れるときは、`latest/download` を `download/v0.2.2` にし、`KIDO_AGENT_VERSION=v0.2.2` を渡します。

取り除くときは `kido-agent uninstall`(操作フォルダ `~/KidoButtons` は残します)。

## 開発

本体とのやり取りは [PROTOCOL.md](PROTOCOL.md) が取り決めです。自分で操作を足すときの見本は [`examples/`](examples/) にあります。

```sh
go vet ./...
go test ./...
# 全部の OS・CPU 向け(cgo は使わない)
for t in windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
  GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go build ./...
done
```

手元のビルドから入れるときは、`KIDO_AGENT_ZIP`(Windows)・`KIDO_AGENT_TARBALL`(Mac・Linux)に zip・tar.gz を渡して、`install/` のスクリプトを動かします(使い方は各スクリプトの先頭)。
CI(`.github/workflows/ci.yml`)は 3 つの OS で、インストールから取り除くまでを実物で通します。

### リリース

`v` で始まるタグを push すると、`release.yml` が配る物(Windows は zip、Mac・Linux は tar.gz)と `SHA256SUMS`、入れ方のスクリプト(`install.ps1`・`install.sh`)を GitHub Releases に上げます。

```sh
git tag v0.2.2 && git push origin v0.2.2
```
