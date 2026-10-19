#!/bin/sh
# 再起動する。-i は、再起動を引き止めるアプリがあっても進めるため
# (押す前に起動ページで確認を出している)。
exec systemctl reboot -i
