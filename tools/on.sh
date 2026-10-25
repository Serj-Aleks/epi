#!/bin/sh
# EPI: вернуть то, что отключил off.sh
O=$HOME/epi-off
[ -e "$O/epi-deskbar" ] && mv "$O/epi-deskbar" "$HOME/config/settings/launch/" && echo "on: epi-deskbar"
[ -e "$O/menu_entries" ] && mv "$O/menu_entries" "$HOME/config/settings/deskbar/" && echo "on: menu_entries"
[ -e "$O/menu_top" ] && mv "$O/menu_top" "$HOME/config/settings/deskbar/" && echo "on: menu_top"
echo "Готово. Перезагрузка."
