# Deskbar EPI: кнопка Θ

Вместо пера Haiku на кнопке меню Deskbar стоит Θ (знак APOTHEOS) того же размера, что и иконки приложений, по центру кнопки. Если в настройках Deskbar изменить размер иконок, Θ подстраивается сама.

## Файлы

- `deskbar-theta.patch` — правка трёх файлов исходников Haiku относительно эталонного коммита `874f0cea9d`: `icons.rdef` (векторная иконка Θ, HVIF), `BarMenuBar.cpp` (квадратная иконка 128×128), `BarMenuTitle.cpp` (размер и центровка).
- `epi-deskbar` — настройка launch_daemon: запускать Deskbar EPI вместо системного.

## Сборка (Debian)

```
cd ~/haiku-dev/haiku
git apply <путь к репозиторию epi>/deskbar/deskbar-theta.patch
cd generated.x86_64
jam -q -j10 Deskbar
```

Результат: `generated.x86_64/objects/haiku/x86_64/release/apps/deskbar/Deskbar`.

## Установка на установленную Haiku

Системный Deskbar не меняется. Файл кладётся под новым именем и переименовывается: перезапись файла работающего Deskbar роняет его.

```
mkdir -p ~/config/non-packaged/apps ~/config/settings/launch
cp Deskbar ~/config/non-packaged/apps/Deskbar.new
mimeset -f ~/config/non-packaged/apps/Deskbar.new
mv ~/config/non-packaged/apps/Deskbar.new ~/config/non-packaged/apps/Deskbar
cp epi-deskbar ~/config/settings/launch/epi-deskbar
```

Затем перезагрузка.

## Откат

Удалить `~/config/settings/launch/epi-deskbar` и перезагрузиться.

## Проверено

24.10.2026, Lenovo, Haiku hrev59866+79, полоса внизу, иконки 40 точек.
