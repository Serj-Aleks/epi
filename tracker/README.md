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

Системная библиотека не меняется: Haiku сначала ищет библиотеки в `~/config/non-packaged/lib`.

```
mkdir -p ~/config/non-packaged/lib
cp libtracker.so ~/config/non-packaged/lib/
hey Tracker quit
```

Tracker перезапускается сам с новой библиотекой. Deskbar подхватит её при следующей загрузке.

## Откат

```
rm ~/config/non-packaged/lib/libtracker.so
hey Tracker quit
```
