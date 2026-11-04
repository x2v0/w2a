# w2a (Web-to-App)

Легковесный инструмент для упаковки статических HTML/JS-сайтов в нативные Android APK.

**Философия:** никаких React, Gradle, Capacitor. Только голый WebView, Go-сборщик и патчинг манифестов. Сборка — **6 секунд**. Размер APK — **37–77 КБ**.

## 🧅 Архитектура

```
w2a/
├── bin/wta                    # Go CLI — сборщик
├── shells/go-webview/         # Нативная оболочка
│   └── shell_webview.apk      # 33 КБ, ноль разрешений
├── templates/                 # Примеры (x2v0-lab, x2v0-resume)
└── Makefile
```

## 🚀 Быстрый старт

```bash
# 1. Установить Go и apktool
pkg install golang apktool

# 2. Собрать ядро
cd ~/w2a && go build -o bin/wta ./cmd/wta

# 3. Собрать шаблон
cd ~/w2a/templates/x2v0-lab && make build
# → build/x2v0-lab.apk (~54 КБ)
```

## 📦 Что внутри

| Компонент | Размер | Роль |
|---|---|---|
| `bin/wta` | 12 МБ | Go CLI: распаковка APK, патч манифеста, влив HTML, подпись |
| `shell_webview.apk` | 33 КБ | Оболочка: ноль разрешений, Java-мост `AndroidPrint` |
| `assets/html/` | зависит | Только `index.html` + нужные ресурсы (мусор вырезан) |

## 🛠 Команды `wta`

```bash
wta build -c wta.yaml               # Собрать APK
wta inspect file.apk --permissions  # Показать разрешения
wta validate -c wta.yaml            # Проверить конфиг
```

## 🔐 Безопасность

- **Ноль разрешений по умолчанию** (даже `INTERNET`).
- Права добавляются явно в `wta.yaml` через `permissions: add:`.
- `MANAGE_EXTERNAL_STORAGE` вырезан из Java-кода навсегда.
- Работает в песочнице Android — не читает пользовательские файлы.

## 🖨 Печать в PDF

Внутри `shell_webview` есть Java-мост `AndroidPrint`. В HTML достаточно:

```javascript
(function() {
    var op = window.print;
    window.print = function() {
        if (window.AndroidPrint && window.AndroidPrint.print) {
            window.AndroidPrint.print();
        } else { op(); }
    };
})();
```

Тогда `window.print()` открывает системный диалог Android с выбором «Сохранить как PDF».

## 🎯 Где используется

- **[Resume Factory](https://github.com/x2v0/resume-factory)** — 17 шаблонов резюме (37–77 КБ).
- **[x2v0-lab](templates/x2v0-lab)** — лаборатория 8 методов загрузки данных.
- **[x2v0-resume](templates/x2v0-resume)** — «святая корова», личное резюме (73 КБ).

## 📋 История версий

| Тег | Что |
|---|---|
| `v2.0-split-projects` | Разделение на x2v0-lab и x2v0-resume |
| `v2.5-zero-perms` | Ноль разрешений, документация |
| `v2.8-strip-shell` | Удалена 74-КБ дефолтная иконка (101→33 КБ) |
| `v2.9-slim-assets` | Вырезаны дубликаты иконок и мусор из APK |
| `v3.0-slim-lab` | x2v0-lab починен, все APK 37–77 КБ |

## 🛡 Лицензия

MIT. Используйте, форкайте, модифицируйте.