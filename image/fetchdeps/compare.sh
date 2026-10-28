#!/bin/sh
# EPI, временно: сравнить image/fetch-deps (Python) и image/fetchdeps (Go)
# на одном и том же собранном образе. Каждая версия работает со своей копией
# epi-local; настоящая epi-local и образ не меняются.
# Запуск на машине сборки: sh ~/haiku-dev/epi/image/fetchdeps/compare.sh
command -v go >/dev/null || { echo "ИТОГ: go не установлен"; exit 1; }
D=$HOME/haiku-dev
C=$HOME/fdcmp
rm -rf "$C"
for v in py go; do
	mkdir -p "$C/$v/haiku-dev"
	ln -s "$D/haiku" "$C/$v/haiku-dev/haiku"
	ln -s "$D/epi" "$C/$v/haiku-dev/epi"
	mkdir -p "$C/$v/haiku-dev/epi-local/epi-packages"
	# пакеты — жёсткими ссылками (место не тратится, файлы не меняются),
	# epi-packages.jam — копией: его перезаписывают
	[ -d "$D/epi-local/epi-packages" ] && cp -al "$D/epi-local/epi-packages/." "$C/$v/haiku-dev/epi-local/epi-packages/"
	[ -f "$D/epi-local/epi-packages.jam" ] && cp "$D/epi-local/epi-packages.jam" "$C/$v/haiku-dev/epi-local/"
done
(cd "$D/epi/image/fetchdeps" && go build -o "$C/fetchdeps" .) || { echo "ИТОГ: Go-версия не собралась"; exit 1; }
HOME=$C/py python3 "$D/epi/image/fetch-deps" > "$C/py.out" 2>&1
HOME=$C/go "$C/fetchdeps" > "$C/go.out" 2>&1
for v in py go; do
	sed "s|$C/$v|H|g" "$C/$v/haiku-dev/epi-local/epi-packages.jam" > "$C/$v.jam" 2>/dev/null
	ls "$C/$v/haiku-dev/epi-local/epi-packages" > "$C/$v.files"
done
r=0
cmp -s "$C/py.out" "$C/go.out" || { echo "вывод различается:"; diff "$C/py.out" "$C/go.out"; r=1; }
cmp -s "$C/py.jam" "$C/go.jam" || { echo "epi-packages.jam различается:"; diff "$C/py.jam" "$C/go.jam"; r=1; }
cmp -s "$C/py.files" "$C/go.files" || { echo "набор пакетов различается:"; diff "$C/py.files" "$C/go.files"; r=1; }
tail -3 "$C/go.out"
if [ $r = 0 ]; then echo "ИТОГ: СОВПАДАЕТ"; rm -rf "$C"; else echo "ИТОГ: РАЗЛИЧАЕТСЯ (подробности в $C)"; fi
