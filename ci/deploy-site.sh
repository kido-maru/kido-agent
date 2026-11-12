#!/bin/sh
# 配布のページ(案内と入れ方のスクリプト)を Cloudflare Workers に手で出す。
#   sh ci/deploy-site.sh                  → pc.kido.page
#   sh ci/deploy-site.sh <ドメイン> <名前>  → 試験用の出し先(ドメインはリポジトリに書かない)
# 配る物(zip・tar.gz)は載せない。入れ方のスクリプトが GitHub Releases から直接落とすので、
# ページだけを自由に出し直せる。
set -eu
domain=${1:-pc.kido.page}
name=${2:-kido-agent-site}
base=https://$domain
out=.site

rm -rf "$out"
mkdir -p "$out"
cp site/index.html site/app.js site/style.css site/_headers "$out/"
cp install/install.ps1 install/install.sh "$out/"
# 入れ方の本物は install/ に 1 つだけ置き、出し先の URL だけをここで書き換える。
# スクリプトが自分を落とし直すときに、出した場所を向くように
for f in index.html install.ps1 install.sh; do
	sed "s#https://pc\.kido\.page#$base#g" "$out/$f" >"$out/$f.tmp"
	mv "$out/$f.tmp" "$out/$f"
done

npx --yes wrangler@4 deploy --name "$name" --domain "$domain"
