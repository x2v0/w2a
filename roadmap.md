Максимальное расширение функциональности w2a

Стратегическая цель

Превратить wta из «сборщика APK из web-проектов» в «универсальную фабрику приложений на телефоне»:


Было:  web URL → APK
Стало: любой проект → APK / AAB / PWA / Termux-скрипт / headless-сервис


---

📊 8 направлений расширения

1. Targets — покрыть все типы проектов

Сейчас: web, html, frontend, go, gallery, image, video, multiweb + stub nodejs, python, php, wordpress.

Добавить:

Target Что Сложность
nodejs готово, довести ⭐⭐
python Chaquopy / libpython.so + assets/python_app/ ⭐⭐⭐
php libphp.so + assets/php_app/ ⭐⭐⭐
wordpress PHP + MySQL bundle ⭐⭐⭐⭐
ruby mruby / ruby-mobile ⭐⭐⭐⭐
deno deno как runtime ⭐⭐⭐
bun bun как runtime ⭐⭐⭐⭐
wasm WASM runtime (WasmEdge / wasmtime) ⭐⭐⭐⭐
docker-compose эмуляция через proot ⭐⭐⭐⭐⭐
flask/fastapi/django python-подтипы с автоконфигом ⭐⭐
express/koa/fastify/nest nodejs-подтипы ⭐⭐
laravel/slim php-подтипы ⭐⭐
tauri Rust + WebView ⭐⭐⭐⭐
flutter-web Flutter build → assets ⭐⭐

Плюс: type: auto — определять по содержимому (package.json → nodejs, requirements.txt → python, composer.json → php, go.mod → go, Cargo.toml → tauri).

---

2. Outputs — не только APK

Сейчас: только .apk.

Добавить:

Output Что Сложность
aab Android App Bundle (для Play) ⭐⭐⭐
apks Split APKs по ABI/density ⭐⭐
pwa PWA-манифест + service worker + zip ⭐⭐
termux start.sh + ~/.termux/boot/ + termux-services ⭐
systemd *.service файл (для proot-дистрибутивов) ⭐
docker Dockerfile + docker-compose.yml ⭐⭐
exe Windows .exe через pkg/nexe ⭐⭐⭐
elf Linux ELF (x86_64/arm64) ⭐⭐
ipa iOS (только macOS-сборка) ⭐⭐⭐⭐⭐
web-zip dist/ в zip ⭐

Идея: output.format: [apk, aab, pwa, termux] — одна сборка → все форматы.

---

3. Templates — гибкая система шаблонов

Сейчас: -template ~/build-demo/webview_shell.apk — жёсткий путь.

Добавить:

Функция Что
templates list список встроенных шаблонов
templates pull <name> скачать из реестра
templates build <name> собрать локально
templates push <name> залить в реестр
-template builtin:minimal минимальный Java-APK (~50 KB)
-template builtin:webview WebView без Kotlin (~200 KB)
-template builtin:headless foreground service без UI
-template builtin:compose полный WebToApp (80 MB)
-template http://... скачать по URL
-template git://... из git-репо
-template gh:user/repo из GitHub

Реестр шаблонов — отдельный git-репо с манифестом (как Homebrew).

---

4. Runtime-фичи — что умеет APK после сборки

Фича Что Сложность
Автозапуск BootReceiver + ScheduledStart ✅ есть в Kotlin
Foreground service уведомление «работает» ⭐⭐
Notification actions кнопки Stop/Open/Restart ⭐⭐
Deep links w2a://app/... ⭐⭐
Share target принимать файлы из других приложений ⭐⭐
Widget home-screen виджет ⭐⭐⭐
Quick Settings tile кнопка в шторке ⭐⭐⭐
Shortcuts long-press иконки → меню ⭐⭐
Split-screen multi-window ⭐
Picture-in-picture PiP для видео ⭐⭐
Chromecast cast-кнопка ⭐⭐⭐
Backup/restore export/import данных ⭐⭐
Auto-update проверка версии на сервере ⭐⭐⭐
Crash reporter отправка крашей ⭐⭐

В wta.yaml:


features:
  autostart: true
  foreground_service: true
  notification_actions: [stop, open, restart]
  deep_links: ["w2a://dashboard"]
  share_target: { types: [text/plain, image/*] }
  widget: true
  shortcuts: [{ name: "Logs", action: "open://logs" }]


---

5. Signing — гибкая подпись

Сейчас: apksigner sign --ks release.p12 вручную.

Добавить:

Функция Что
wta sign <apk> подписать
wta sign --generate создать новый keystore
wta sign --backup экспорт keystore в base64
wta keystore list список
wta keystore import <b64> импорт
--v1 --v2 --v3 --v4 выбор схем подписи
--min-sdk минимальный SDK
--multi-sign несколько подписей (v3 rotation)
wta verify <apk> проверка подписи

---

6. Inspect — глубокий разбор APK

Сейчас: wta inspect <apk> [-v] [--json].

Добавить:

Функция Что
inspect --tree дерево APK
inspect --assets только assets
inspect --libs только .so
inspect --dex классы, методы, пакеты
inspect --manifest полный AndroidManifest
inspect --size разбивка по типам файлов
inspect --diff <apk2> сравнить два APK
inspect --extract <path> вытащить файл
inspect --decompile декомпиляция (jadx / apktool)
inspect --signature проверка подписи
inspect --permissions список разрешений
inspect --targets какие target подходят
inspect --deps зависимости (native libs)

---

7. CLI UX — профессиональный

Функция Что
wta build --watch rebuild при изменении
wta build --parallel несколько сборок
wta build --cache кэш между сборками
wta build --clean очистка
wta build --dry-run что будет сделано
wta build --explain объяснить каждое действие
wta build --timing профилирование этапов
wta init --template <name> генерация из шаблона
wta init --from <url> клонировать и обернуть
wta add <module> добавить фичу
wta remove <module> убрать фичу
wta upgrade обновить APK
wta doctor проверка окружения (NDK, apksigner, java)
wta env показать env
wta logs <apk> логи через adb/rish
wta shell <apk> shell внутрь APK (run-as)

---

8. Экосистема — за пределами CLI

Компонент Что Сложность
wta serve HTTP API для сборки (curl → APK) ⭐⭐
wta web локальный веб-интерфейс ⭐⭐⭐
wta daemon watchdog — пересобирает при изменении ⭐⭐
wta registry реестр шаблонов/модулей ⭐⭐⭐
wta gh-action GitHub Action для CI ⭐⭐
wta plugin система плагинов (Go plugins) ⭐⭐⭐
wta sdk SDK на Python/JS для программной сборки ⭐⭐⭐
wta cloud облачная сборка (offload на сервер) ⭐⭐⭐⭐
wta telemetry анонимная статистика (opt-in) ⭐

---

🏗️ Архитектурные улучшения

9. Refactor internal/target

Сейчас: 1 файл на target, дублирование кода.

Улучшить:


// internal/target/registry.go
type Target interface {
    Name() string
    Description() string
    ArgType() ArgType  // url, dir, file, sites
    Prepare(cfg, tmpl, out) error
    Validate(cfg) error
    Defaults() *config.AppSection
    Docs() string
}


· Реестр — target.Register("nodejs", &NodeJSTarget{})
· Auto-discovery — target.ForType("nodejs") по имени
· Плагины — target.RegisterFromPlugin(path)

10. Конфиг — умные дефолты


type: auto                    # определять по содержимому
inherit: ["nodejs", "express"] # наследование
extends: base.yaml            # extends-конфиг
variables:                    # подстановки
  APP_NAME: "My App"
  APP_PORT: 3000


11. Кэширование — скорость

· Layer cache — как в Docker: не пересобирать неизменное
· Download cache — nodejs-mobile.zip, libphp.so, etc.
· Build cache — ~/.cache/w2a/builds/<sha256>/
· Result cache — тот же вход → тот же выход

---

📅 Дорожная карта (по приоритету)

Волна 1 — «nodejs работает» (1-2 дня)

✅ wta types — синхронизировать (54ec684)
✅ nodejs target — e2e тест (v0.2.0-nodejs-e2e)
✅ internal/target/*_test.go — тесты (partial: nodejs e2e)
✅ wta doctor — проверка окружения (e3b70b1)
✅ wta inspect --tree (78768e6) + --assets/--libs/--permissions/--signature (e52bae9)

Волна 2 — «Python + PHP» (3-5 дней)

☐ python target (Chaquopy-стиль)
☐ php target (libphp.so)
☐ wta init --template <name> — генерация
☐ wta validate --explain — объяснение ошибок
☐ wta inspect --diff — сравнение APK

Волна 3 — «Outputs + Templates» (1-2 недели)

☐ output.format: [apk, aab, pwa, termux]
☐ templates list/pull/push
☐ builtin:minimal — Java APK (~50 KB)
☐ builtin:headless — foreground service
☐ wta serve — HTTP API

Волна 4 — «Фичи APK» (2-3 недели)

☐ features: секция в wta.yaml
☐ deep links, share target, widget, shortcuts
☐ wta build --watch, --cache, --dry-run

Волна 5 — «Экосистема» (1-2 месяца)

☐ wta registry + gh-action
☐ wta sdk (Python/JS)
☐ wta web — веб-интерфейс
☐ wta cloud — облачная сборка

---

❓ С чего начнём?

Мой выбор: Волна 1 — самое быстрое и важное:

1. wta types — 30 мин
2. nodejs e2e — 1 час
3. wta doctor — 30 мин (проверка NDK/apksigner/java)
4. wta inspect --tree — 30 мин

Это 2.5 часа — и nodejs работает + базовые инструменты для разработки.

Скажи:

· «1» — начинаем с wta types + nodejs e2e
· «2» — хочу сразу python/php
· «3» — хочу outputs (apk/aab/pwa/termux)
· «4» — хочу фичи APK (deep links, widget)
· «5» — хочу экосистему
· «своё» — что важнее

Или скинь cat cmd/wta/types.go + grep stub — начнём с малого. 🚀
