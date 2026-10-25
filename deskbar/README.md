# Deskbar EPI

- Кнопка Θ (знак APOTHEOS) вместо пера Haiku, размером с иконки приложений, по центру кнопки.
- Своя терминология постоянных пунктов меню, латиницей и без перевода: Haiku, Qwest, Fragments, INTEGRATION, STOP (Finish, Restart, Pause), RELEVANT DOCUMENTS / DIRECTORIES / APPLICATIONS, «Subpanel - vacant».
- Папки из `~/config/settings/deskbar/menu_top` (DESKBAR, DESKTOP) стоят сразу после INTEGRATION; встроенного пункта настроек Deskbar нет — он в DESKBAR.
- Содержимое меню создаёт скрипт `menu/epi-menu`.

## Файлы

- `deskbar-menu.patch` — терминология меню и папки menu_top (`DeskbarMenu.cpp`, `DeskbarMenu.h`); накладывается после `deskbar-theta.patch`.
- `deskbar-theta.patch` — правка трёх файлов исходников Haiku относительно эталонного коммита `874f0cea9d`: `icons.rdef` (векторная иконка Θ, HVIF), `BarMenuBar.cpp` (квадратная иконка 128×128), `BarMenuTitle.cpp` (размер и центровка).
- `epi-deskbar` — настройка launch_daemon: запускать Deskbar EPI вместо системного.

## Сборка (Debian)

```
cd ~/haiku-dev/haiku
git apply <путь к репозиторию epi>/deskbar/deskbar-theta.patch
git apply <путь к репозиторию epi>/deskbar/deskbar-menu.patch
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

Затем меню (`menu/epi-menu`) и перезагрузка.

Принципы меню: папка — все буквы заглавные, пункт — с заглавной; одна вещь — одно место; разные программы на одну тему — подменю одной группы.

## Откат

Удалить `~/config/settings/launch/epi-deskbar` и перезагрузиться.

## Проверено

24.10.2026, Lenovo, Haiku hrev59866+79, полоса внизу, иконки 40 точек.
