APK_NAME = x2v0-resume.apk
BUILD_DIR = build
WTA = ~/w2a/bin/wta
CONFIG = wta.yaml

.PHONY: all build clean install

all: build

build:
	@echo "=== Сборка APK (Резюме) ==="
	rm -rf $(BUILD_DIR)
	$(WTA) build -c $(CONFIG)
	cp $(BUILD_DIR)/$(APK_NAME) /sdcard/Download/
	@echo "✓ APK скопирован в /sdcard/Download/"

clean:
	rm -rf $(BUILD_DIR)
