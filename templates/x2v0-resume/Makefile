APK_NAME = x2v0-resume.apk
BUILD_DIR = build
WTA = ~/w2a/bin/wta
CONFIG = wta.yaml

.PHONY: all build clean rebuild help

all: build

help:
	@echo "Доступные команды для x2v0-resume:"
	@echo "  make build    - Собрать APK"
	@echo "  make clean    - Удалить папку build"
	@echo "  make rebuild  - Очистить и собрать заново"
	@echo "  make help     - Показать эту справку"

clean:
	@echo "=== Очистка ==="
	rm -rf $(BUILD_DIR)
	@echo "OK: папка build/ удалена"

build:
	@echo "=== Сборка APK (Резюме) ==="
	@mkdir -p .temp_hidden
	@mv Makefile icon-192.png icon-512.png .temp_hidden/ 2>/dev/null || true
	$(WTA) build -c $(CONFIG)
	@mv .temp_hidden/* . 2>/dev/null || true
	@rmdir .temp_hidden 2>/dev/null || true
	@cp $(BUILD_DIR)/$(APK_NAME) /sdcard/Download/
	@echo "OK: APK скопирован"

rebuild: clean build
