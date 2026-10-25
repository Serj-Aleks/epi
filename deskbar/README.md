# Deskbar EPI

- Кнопка Θ (знак APOTHEOS) вместо пера Haiku, размером с иконки приложений, по центру кнопки.
- Своя терминология постоянных пунктов меню, латиницей и без перевода: Haiku, Qwest, Fragments, INTEGRATION, STOP (Finish, Restart, Pause), RELEVANT (DOCUMENTS, DIRECTORIES, APPLICATIONS), «Directory - vacant» (пустая папка меню и пустые подменю RELEVANT).
- Папки из `~/config/settings/deskbar/menu_top` (DESKBAR, DESKTOP) стоят сразу после INTEGRATION; встроенного пункта настроек Deskbar нет — он в DESKBAR.
- Содержимое меню создаёт скрипт `menu/epi-menu`.
- Запущенные программы в полосе (значки, подсказки, списки окон) называются так же, как их ссылки в меню EPI: Tracker — Navigator, LaunchBox — Panel, StyledEdit — TextEditor и т. д. Одна терминология, источник — `menu/epi-menu`.

## Файлы

- `deskbar-epi.patch` — все правки Deskbar EPI относительно эталонного коммита `874f0cea9d`: `icons.rdef` (иконка Θ, HVIF), `BarMenuBar.cpp`, `BarMenuTitle.cpp` (размер и центровка Θ), `DeskbarMenu.cpp`, `DeskbarMenu.h` (терминология, папки menu_top, RELEVANT), `BarApp.cpp`, `BarApp.h` (имена программ в полосе по меню EPI).
- `epi-deskbar` — настройка launch_daemon: запускать Deskbar и файловый менеджер EPI вместо системных (`~/config/settings/launch/epi-deskbar`).

## Сборка (Debian)

```
cd ~/haiku-dev/haiku
git checkout -- src/apps/deskbar
git apply <путь к репозиторию epi>/deskbar/deskbar-epi.patch
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
