# Examples

10 готовых конфигов `wta.yaml` — от простого к сложному.

## Быстрый старт

1. **Получите template APK** (`shell.apk`).
   - Скачайте из GitHub Actions (`standard-apks-DEBUG-SIGNED-...zip` → `app-standard-debug.apk`)
   - Или соберите: `./gradlew :app:assembleStandardDebug`
   - Укажите путь: `export WTA_TEMPLATE=/path/to/shell.apk`

2. **Соберите бинарник `wta`:**
   ```bash
   cd go && make build
   ```

3. **Запустите пример:**
   ```bash
   cd go/examples
   ../bin/wta build -c 01-web.yaml
   ```

4. **Проверьте результат:**
   ```bash
   ../bin/wta-buildapk -template build/01-web.apk -inspect
   unzip -p build/01-web.apk assets/webtoapp/config.json
   ```

## Список

| # | Файл | Что демонстрирует |
|---|---|---|
| 01 | `01-web.yaml` | Простейший WebView |
| 02 | `02-html.yaml` | HTML-проект из папки |
| 03 | `03-frontend.yaml` | SPA (React/Vue) из dist/ |
| 04 | `04-go.yaml` | Go-сервер (cross-compile) |
| 05 | `05-gallery.yaml` | Галерея изображений |
| 06 | `06-image.yaml` | Одно изображение |
| 07 | `07-video.yaml` | Одно видео |
| 08 | `08-multiweb.yaml` | Несколько сайтов |
| 09 | `09-with-permissions.yaml` | Permissions add/remove |
| 10 | `10-full-featured.yaml` | Всё сразу |

## Env-переменные

| Var | Значение |
|---|---|
| `WTA_TEMPLATE` | Путь к template APK |
| `ANDROID_HOME` | Для `zipalign` (если `align` задан) |
| `ANDROID_NDK_HOME` | Для Go cgo (если `noCgo: false`) |
| `GITHUB_TOKEN` | Для `wta-gensubmissions` |

## Смотрите также

- [../docs/QUICKSTART.md](../docs/QUICKSTART.md) — быстрый старт с нуля
- [../docs/TUTORIAL.md](../docs/TUTORIAL.md) — 8 уроков пошагово
- [../docs/COMMANDS.md](../docs/COMMANDS.md) — справочник команд
