# Tracker (NAVIGATOR) EPI: URNA и DISKS

Файловый менеджер Haiku показывает Корзину как **URNA**, а корень дисков как **DISKS** (латиницей, без перевода). Правка одна: `Model::Name()` в `src/kits/tracker/Model.cpp`, библиотека `libtracker.so`.

## Сборка (Debian)

```
cd ~/haiku-dev/haiku
git checkout -- src/kits/tracker
git apply <путь к репозиторию epi>/tracker/tracker-epi.patch
cd generated.x86_64
jam -q -j10 libtracker.so
```

Результат: `generated.x86_64/objects/haiku/x86_64/release/kits/tracker/libtracker.so`.

## Установка на установленную Haiku

Системные файлы не меняются. Файловый менеджер, лежащий в `/boot/system`, всегда берёт библиотеку из `/boot/system/lib` (первой в поиске стоит папка самой программы, `%A/lib`). Поэтому запускается своя копия файлового менеджера, а рядом с ней, в `lib`, — наша библиотека.

```
mkdir -p ~/config/non-packaged/apps/lib
cp /boot/system/Tracker ~/config/non-packaged/apps/Tracker
mimeset -f ~/config/non-packaged/apps/Tracker
cp libtracker.so ~/config/non-packaged/apps/lib/
```

Запуск копии вместо системного — блок `x-vnd.Be-TRAK` в `deskbar/epi-deskbar`. Deskbar EPI лежит в той же папке и тоже берёт эту библиотеку. Затем перезагрузка.

## Откат

Убрать блок `x-vnd.Be-TRAK` из `~/config/settings/launch/epi-deskbar` и перезагрузиться.
