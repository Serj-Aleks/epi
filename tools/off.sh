#!/bin/sh
# EPI: временно отключить всё, что менялось 24-25.10 (Deskbar EPI, меню EPI).
# Ничего не удаляет: переносит в ~/epi-off. Вернуть: sh on.sh
O=$HOME/epi-off
mkdir -p "$O"
for f in config/settings/launch/epi-deskbar config/settings/deskbar/menu_entries config/settings/deskbar/menu_top; do
	[ -e "$HOME/$f" ] && mv "$HOME/$f" "$O/" && echo "off: $f"
done
echo "Готово. Теперь полное выключение: меню Θ -> STOP -> Finish, 10 секунд, включить."
