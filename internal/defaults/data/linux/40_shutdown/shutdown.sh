#!/bin/sh
# シャットダウンする。-i は、止めるのを引き止めるアプリがあっても進めるため
# (押す前に起動ページで確認を出している)。
exec systemctl poweroff -i
