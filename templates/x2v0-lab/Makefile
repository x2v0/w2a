APK_NAME = x2v0-lab.apk
BUILD_DIR = build
WTA = ~/w2a/bin/wta
CONFIG = wta.yaml

.PHONY: all build clean rebuild base64

all: build

base64:
	@echo "=== Обновление en_base64.js ==="
	@python3 -c "import base64, pathlib, re; \
	html = pathlib.Path('en/index.html').read_text(encoding='utf-8'); \
	html = re.sub(r'>\\s+<', '><', html); \
	pathlib.Path('en/index.html').write_text(html, encoding='utf-8'); \
	b64 = base64.b64encode(html.encode('utf-8')).decode('utf-8'); \
	pathlib.Path('en_base64.js').write_text(f'window.EN_HTML_BASE64 = \"data:text/html;base64,{b64}\";', encoding='utf-8'); \
	print('✓ en_base64.js обновлен')"

clean:
	@echo "=== Очистка ==="
	rm -rf $(BUILD_DIR)
	@echo "✓ Папка build/ удалена"

build: base64
	@echo "=== Сборка APK (Оптимизированная) ==="
	@mkdir -p ../lab_temp_hidden
	@# Прячем мусор, который не нужен в APK, но нужен для проекта
	@mv build-apk.sh sw.js manifest.json icon.svg icon-192.png portfolio ../lab_temp_hidden/ 2>/dev/null || true
	$(WTA) build -c $(CONFIG)
	@# Возвращаем файлы
	@mv ../lab_temp_hidden/* . 2>/dev/null || true
	@rmdir ../lab_temp_hidden 2>/dev/null || true
	@cp $(BUILD_DIR)/$(APK_NAME) /sdcard/Download/
	@echo "✓ APK скопирован в /sdcard/Download/"

rebuild: clean build
