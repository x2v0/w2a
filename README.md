# w2a (Web-to-App)

Легковесная альтернатива Capacitor/Cordova. Упаковывает статические HTML/JS сайты в нативные APK.

## 💡 Философия
Без `node_modules`, React или Gradle. Системный WebView, Go-код и патчинг манифестов. Сборка — **5 секунд**, APK — **98-190 КБ**.

## 🧅 Архитектура
1.  **shell_webview.apk**: Нативная оболочка (Go+Java). Ноль прав по умолчанию. Нативный мост `AndroidPrint`.
2.  **templates/**: HTML-проекты (Резюме, Лаборатория).
3.  **wta (Go CLI)**: Патчит манифест, подписывает APK.

## 🛠 Сборка
```bash
cd shells/go-webview && ./build.sh  # Оболочка
cd templates/x2v0-resume && make    # Резюме
cd templates/x2v0-lab && make       # Лаборатория
```

## 🛡 Безопасность
Ноль прав по умолчанию. Права добавляются явно через `permissions: add:` в `wta.yaml`.
