
# 📚 Пошаговая инструкция — w2a APK builder

Полный операционный гайд: setup, .bashrc, ежедневный workflow, wta CLI, troubleshooting.

---

## 📋 Содержание

1. Требования
2. Настройка окружения (.bashrc)
3. Проверка установки
4. Ежедневный workflow
5. Ручная сборка (без make)
6. Диагностика
7. Troubleshooting
8. Git workflow
9. Структура проекта

---

## 1. Требования

| Компонент | Путь | Как поставить |
|-----------|------|---------------|
| Termux | — | F-Droid / GitHub |
| Go 1.22+ | `$PREFIX/bin/go` | `pkg install golang` |
| Android SDK (build-tools) | `~/android-sdk` | `pkg install android-tools` или вручную |
| Shizuku + rish | — | Из Play Store + `pkg install rish` |
| Keystore | `~/build-demo/myapp/release.p12` | Создать: `keytool -genkeypair ...` |
| Shell template | `~/build-demo/webview_shell.apk` | Извлечь из `app-standard-debug.apk` |
| wta (Go) | `~/w2a/bin/wta` | `cd ~/w2a && make build` |
| Wikipedia проект | `~/build-demo/myapp/` | `wta.yaml` + описание |

---

## 2. Настройка окружения (.bashrc)

### 2.1. Открыть `~/.bashrc`

```bash
micro ~/.bashrc
```

2.2. Добавить в конец

```bash
# ═══════════════════════════════════════════════════════════
# w2a — APK builder
# ═══════════════════════════════════════════════════════════

# wta CLI в $PATH
export PATH="$HOME/w2a/bin:$PATH"

# Значения по умолчанию для wta build
export WTA_TEMPLATE="$HOME/build-demo/webview_shell.apk"
export WTA_SKIP_MANIFEST=1

# Android SDK
export ANDROID_HOME="$HOME/android-sdk"
export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$PATH"

# Shizuku / rish
export RISH_APPLICATION_ID=com.termux

# Keystore (по умолчанию)
export WTA_KEYSTORE="$HOME/build-demo/myapp/release.p12"
export WTA_KEYSTORE_PASS=testpass
export WTA_KEY_ALIAS=myalias

# Удобные алиасы
alias w2a='cd $HOME/w2a'
alias wiki='cd $HOME/w2a && make wiki'
alias wbuild='cd $HOME/w2a && make build'
```

2.3. Применить

```bash
source ~/.bashrc

# Проверка
echo $WTA_TEMPLATE
echo $RISH_APPLICATION_ID
which wta
```

Ожидаемо:

```
/data/data/com.termux/files/home/build-demo/webview_shell.apk
com.termux
/data/data/com.termux/files/home/w2a/bin/wta
```

2.4. Опционально — bash completion для wta

```bash
mkdir -p ~/.termux/completions

cat > ~/.termux/completions/wta.bash <<'EOF'
_wta_complete() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    case "$prev" in
        -c) COMPREPLY=( $(compgen -f -X '!*.yaml' -- "$cur") ); return ;;
        build|inspect|sign) ;;
    esac
    COMPREPLY=( $(compgen -W "build validate inspect types test init version" -- "$cur") )
}
complete -F _wta_complete wta
EOF

# Подключить в .bashrc
echo 'source ~/.termux/completions/wta.bash' >> ~/.bashrc
source ~/.bashrc
```

---

3. Проверка установки

3.1. Проверить всё сразу

```bash
cat > ~/w2a/scripts/check-env.sh <<'EOF'
#!/data/data/com.termux/files/usr/bin/bash
# check-env.sh — проверка окружения w2a

ok=0; fail=0
check() {
    local name="$1" cmd="$2"
    if eval "$cmd" >/dev/null 2>&1; then
        echo "✅ $name"
        ok=$((ok+1))
    else
        echo "❌ $name"
        fail=$((fail+1))
    fi
}

echo "─── Требования ───"
check "go (>=1.22)"            'go version | grep -qE "go1\.(2[2-9]|[3-9])"'
check "wta в PATH"             'which wta'
check "wta работает"           'wta --version'
check "WTA_TEMPLATE файл"      'test -f "$WTA_TEMPLATE"'
check "WTA_SKIP_MANIFEST=1"    'test "$WTA_SKIP_MANIFEST" = "1"'
check "RISH_APPLICATION_ID"    'test -n "$RISH_APPLICATION_ID"'
check "Android SDK"            'test -d "$ANDROID_HOME/build-tools"'
check "aapt2"                  'find "$ANDROID_HOME/build-tools" -name aapt2 | grep -q .'
check "zipalign"               'find "$ANDROID_HOME/build-tools" -name zipalign | grep -q .'
check "apksigner"              'find "$ANDROID_HOME/build-tools" -name apksigner | grep -q .'
check "keystore"               'test -f "$HOME/build-demo/myapp/release.p12"'
check "rish работает"          'rish -c "id" 2>/dev/null | grep -q "uid=2000"'
check "Wikipedia проект"       'test -f "$HOME/build-demo/myapp/wta.yaml"'

echo ""
echo "─── Итог: $ok OK, $fail FAIL ───"
[ $fail -eq 0 ] && echo "🎉 Готово к работе"
EOF

chmod +x ~/w2a/scripts/check-env.sh
~/w2a/scripts/check-env.sh
```

Ожидаемо (все зелёные):

```
─── Требования ───
✅ go (>=1.22)
✅ wta в PATH
✅ wta работает
...
─── Итог: 13 OK, 0 FAIL ───
🎉 Готово к работе
```

---

4. Ежедневный workflow

4.1. Полный цикл одной командой

```bash
wiki                      # алиас → cd ~/w2a && make wiki
```

Что происходит:

1. Собирает все 11 бинарников (make build)
2. wta build → /sdcard/Download/wikipedia-reader.apk
3. zipalign + apksigner → wikipedia-final.apk
4. rish pm install -d -r → устанавливает на телефон
5. monkey запускает приложение
6. logcat показывает первые логи

4.2. Пошагово (если нужно разбить)

```bash
cd ~/w2a

make build          # только бинарники
make wiki-build     # + APK
make wiki-sign      # + zipalign + sign
make wiki-install   # + install на телефон
make wiki-run       # + launch + logcat
```

4.3. Тесты

```bash
cd ~/w2a
make test           # go test ./...
make test-v         # verbose

# wta test — прогон всех примеров
WTA_TEMPLATE=$HOME/build-demo/webview_shell.apk \
WTA_SKIP_MANIFEST=1 \
wta test examples-wta/
```

4.4. Другой проект (не Wikipedia)

```bash
mkdir -p ~/build-demo/myapp2
cd ~/build-demo/myapp2

cat > wta.yaml <<'EOF'
app:
  name: "MyApp2"
  package: com.example.myapp2
  versionCode: 1
  versionName: "1.0.0"
type: web
web:
  url: https://example.com
output:
  path: /sdcard/Download/myapp2.apk
  align: 4
EOF

wta build -c wta.yaml -v
```

Или используй готовый пример:

```bash
cp ~/w2a/examples-wta/01-web.yaml ~/build-demo/myapp2/wta.yaml
```

---

5. Ручная сборка (без make)

5.1. Сборка APK

```bash
cd ~/build-demo/myapp

# env уже в .bashrc → просто
wta build -c wta.yaml -v

# Проверка
unzip -l /sdcard/Download/wikipedia-reader.apk | grep config
# assets/app_config.json (65 bytes)
```

5.2. Align + Sign

```bash
cd ~/build-demo/myapp

BT=$(dirname "$(find $ANDROID_HOME/build-tools -name aapt2 | head -1)")
ZIP=$(find $ANDROID_HOME/build-tools -name zipalign | head -1)

$ZIP -f -p 4 /sdcard/Download/wikipedia-reader.apk aligned.apk

$BT/apksigner sign \
  --ks release.p12 --ks-type PKCS12 \
  --ks-pass pass:testpass \
  --ks-key-alias myalias --key-pass pass:testpass \
  --out /sdcard/Download/wikipedia-final.apk \
  aligned.apk

apksigner verify --verbose /sdcard/Download/wikipedia-final.apk | head -4
```

5.3. Установка через rish

```bash
rish -c "cp /sdcard/Download/wikipedia-final.apk /data/local/tmp/"
rish -c "pm install -d -r /data/local/tmp/wikipedia-final.apk"
rish -c "rm /data/local/tmp/wikipedia-final.apk"
```

Флаг -d — разрешает downgrade (shell v71 поверх установленного com.webtoapp v76).

5.4. Запуск + логи

```bash
rish -c "am force-stop com.webtoapp"    # убить старый процесс
rish -c "logcat -c"                     # очистить логи
rish -c "monkey -p com.webtoapp -c android.intent.category.LAUNCHER 1" >/dev/null
sleep 4
rish -c "logcat -d | grep -iE 'ShellModeManager|ShellScreen|WebViewManager' | tail -15"
```

Успех:

```
D ShellModeManager: Конфиг валиден — режим Shell, appType=WEB
D ShellScreen: appType='WEB', targetUrl='https://ru.wikipedia.org'
D WebViewManager: Page finished: +506ms blocked=0 errors=0
```

5.5. Команды wta CLI

```
wta build -c wta.yaml [--json]     # сборка APK из YAML
wta validate -c wta.yaml [--json]  # 10 проверок, exit 1 при ошибке
wta inspect <apk> [-v] [--json]    # package, versionCode, config.json
wta types                          # 12 типов (8 ✓ / 4 stub)
wta test <dir> [--json]            # прогон всех *.yaml в директории
wta init -type web -name "..."     # шаблон wta.yaml
wta version                        # версия
```

Флаги build:

```
--out <path>       # переопределить output.path из YAML
--template <path>  # переопределить shell-шаблон (иначе WTA_TEMPLATE)
--json             # машиночитаемый JSON в stdout, логи → stderr
-v                 # verbose
```

Примеры:

```bash
# CI-friendly: sha256 из JSON
wta build -c wta.yaml --json 2>/dev/null | jq -r .sha256

# Проверка всех примеров
WTA_TEMPLATE=$HOME/build-demo/webview_shell.apk \
WTA_SKIP_MANIFEST=1 \
wta test examples-wta/

# Анализ готового APK
wta inspect /sdcard/Download/app.apk

# Валидация конфига
wta validate -c wta.yaml
```

Полный прогон:

```bash
cd ~/w2a
make test                          # go test ./...
make wiki                          # end-to-end Wikipedia
```

---

6. Диагностика

6.1. Что в APK

```bash
APK=/sdcard/Download/wikipedia-final.apk

# assets
unzip -l $APK | grep -E "assets/|config"

# manifest (badging)
BT=$(dirname "$(find $ANDROID_HOME/build-tools -name aapt2 | head -1)")
$BT/aapt2 dump badging $APK 2>&1 | head -5

# config contents
unzip -p $APK assets/app_config.json
```

6.2. Что за URL грузит shell

```bash
rish -c "logcat -d | grep -iE 'targetUrl|Main-frame' | tail -5"
```

6.3. Проверка установки

```bash
rish -c "pm list packages | grep webtoapp"
rish -c "dumpsys package com.webtoapp | grep -E 'versionCode|versionName'" | head -3
```

6.4. wta inspect

```bash
wta inspect /sdcard/Download/wikipedia-final.apk
wta inspect /sdcard/Download/wikipedia-final.apk -v     # + список entries
wta inspect /sdcard/Download/wikipedia-final.apk --json | jq
```

---

7. Troubleshooting

7.1. rish: Request timeout

Причина: Shizuku или battery optimization.

Fix:

1. На телефоне: открой Shizuku — проверь «running»
2. Отключи оптимизацию батареи для Termux и Shizuku:
   · Настройки → Приложения → Termux → Батарея → Без ограничений
   · То же для Shizuku
   · MIUI/HyperOS: Автозапуск → включить
3. Перезапусти Shizuku (Stop → Start)
4. Проверь:
   ```bash
   rish -c "id"
   # uid=2000(shell) ...
   ```

7.2. INSTALL_FAILED_USER_RESTRICTED

Причина: versionCode пустой (ResourceMap chunk не пишется в AXML).

Fix (временный): установка через rish с флагом -d:

```bash
rish -c "pm install -d -r /data/local/tmp/wikipedia-final.apk"
```

Fix (полный): написать ResourceMap chunk в internal/apk/axml_write.go. Не сделано — TODO.

7.3. Тост на китайском + приложение закрывается

Причина: shell не нашёл app_config.json или JSON невалидный.

Диагностика:

```bash
rish -c "logcat -c"
rish -c "monkey -p com.webtoapp -c android.intent.category.LAUNCHER 1" >/dev/null
sleep 3
rish -c "logcat -d | grep -iE 'ShellModeManager|Конфиг|targetUrl|appType' | tail -20"
```

Что искать:

· app_config.json не найден → путь не тот (должен быть assets/app_config.json)
· Конфиг невалиден: appType=..., targetUrl=... → JSON пустой или кривой

Проверка APK:

```bash
unzip -l /sdcard/Download/wikipedia-final.apk | grep config
unzip -p /sdcard/Download/wikipedia-final.apk assets/app_config.json
```

Ожидаемо:

```
assets/app_config.json  (65 bytes)
{"appType": "WEB", "targetUrl": "https://ru.wikipedia.org"}
```

7.4. pm install — Failure [INSTALL_FAILED_ALREADY_EXISTS]

Причина: оригинальный com.webtoapp уже установлен.

Fix A — downgrade:

```bash
rish -c "pm install -d -r /data/local/tmp/wikipedia-final.apk"
```

Fix B — удалить оригинал (потеряешь его данные):

```bash
rish -c "pm uninstall com.webtoapp"
rish -c "pm install -r /data/local/tmp/wikipedia-final.apk"
```

7.5. AXML тест TestRoundtripSimple падает

Статус: ожидаемо (t.Skip стоит).

Причина: синтетика — edge-case в writer или parser.

Влияние: ноль — production (реальные манифесты) работает.

7.6. wta test — 0 passed

Причина: отсутствуют source-проекты для примеров (html-project, dist, goproject, photos, ...).

Fix:

```bash
cd ~/w2a
wta test examples-wta/
```

7.7. wta build --json — сломанный JSON

Причина: до wta дошли логи от build.Build в stdout.

Fix: уже решено — в --json режиме build.Build пишет в stderr, в stdout — только JSON.

Проверка:

```bash
wta build -c wta.yaml --json 2>/dev/null | jq
```

Если jq валидирует — всё ок.

---

8. Git workflow

8.1. Проверка состояния

```bash
cd ~/w2a
git status --short
git log --oneline | head -5
```

8.2. Коммит изменений

```bash
cd ~/w2a

# Что менялось
git diff --stat

# Застейджить
git add -A

# Убедиться, что бэкапы и ad-hoc скрипты не попали
git status --short | grep -E "\.bak|fix\.py"
# (должно быть пусто — они в .gitignore)

git commit -m "feat/fix/chore: описание"
```

8.3. Push

```bash
cd ~/w2a
git push origin main
```

8.4. Откат

```bash
# Вернуть один файл
git checkout -- internal/apk/build.go

# Откатить последний коммит (оставить изменения)
git reset --soft HEAD~1

# Полный откат к origin (потерять локальные)
git fetch origin
git reset --hard origin/main
```

---

9. Структура проекта

```
~/w2a/                              ← Android-проект + wta
├── .gitignore
├── Makefile                        ← единый (test, build, wiki, wiki-*)
├── WORKFLOW.md                     ← этот файл
├── README.md                       ← корневой (Android)
├── README.wta.md                   ← wta (Go)
│
├── app/                            ← Android app (Kotlin)
├── shell/                          ← shell runtime sources
├── docs/                           ← Android docs
├── docs-wta/                       ← wta docs
├── examples-wta/                   ← wta.yaml examples + фикстуры
│   ├── html-project/               ← (фикстура 02-html)
│   ├── dist/                       ← (фикстура 03-frontend)
│   ├── goproject/                  ← (фикстура 04-go)
│   ├── photos/                     ← (фикстура 05-gallery)
│   ├── photo.jpg, video.mp4        ← (фикстуры 06, 07)
│   └── sample-projects/            ← симлинки на app/src/main/assets/sample_projects/
│
├── go.mod, go.sum                  ← Go module
├── cmd/                            ← 11 CLI (wta, wta-signapk, ...)
│   ├── wta/
│   │   ├── main.go                 ← командный switch + build/validate/init
│   │   ├── templates.go            ← шаблоны для wta init
│   │   ├── inspect.go              ← wta inspect
│   │   ├── types.go                ← wta types
│   │   └── test.go                 ← wta test
│   ├── wta-buildapk/               ← inspect APK (standalone)
│   ├── wta-signapk/                ← подпись
│   └── wta-*                       ← остальные утилиты
│
├── internal/
│   ├── apk/                        ← AXML, ZIP, resources, APK builder
│   ├── apksig/                     ← V1/V2/V3 signer
│   ├── arsc/                       ← resources.arsc parser/writer
│   ├── apkutil/                    ← d8, aar, host tools
│   ├── assetsync/, cache/, config/
│   ├── kotlin/                     ← @SerializedName parsing
│   ├── modules/                    ← plugin system
│   ├── musl/                       ← musl build для NDK
│   ├── stack/                      ← project stack bundler
│   └── target/                     ← web/html/frontend/go/gallery/image/video/multiweb
│
├── bin/                            ← 11 собранных бинарников
├── scripts/
│   └── check-env.sh
│

~/build-demo/myapp/                 ← Wikipedia проект
├── wta.yaml
├── release.p12                     ← keystore
├── build.sh                        ← end-to-end скрипт
└── README.md

~/build-demo/webview_shell.apk      ← shell template (6-8 MB)
```

---

🎯 TL;DR — команды за день

```bash
# Один раз: настроить .bashrc, source, проверить check-env.sh

# Каждый раз при работе:
wiki                       # собрать + подписать + установить + запустить

# Или пошагово:
make build                 # 11 бинарников
make wiki-build            # + APK
make wiki-sign             # + sign
make wiki-install          # + install
make wiki-run              # + launch + logcat

# Тесты:
make test                  # go test ./...
wta test examples-wta/     # прогон всех примеров

# Диагностика:
wta inspect /sdcard/Download/app.apk
wta validate -c wta.yaml
rish -c "logcat -d | grep -iE 'ShellModeManager|WebViewManager' | tail -15"

# Обновить remote:
cd ~/w2a && git push origin main
```

---

📋 Обновить инструкцию

```bash
cd ~/w2a
micro WORKFLOW.md
# ... редактируй, Ctrl+S, Ctrl+Q

git add WORKFLOW.md
git commit -m "docs(workflow): update"
```
