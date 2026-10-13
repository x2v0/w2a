# ═══════════════════════════════════════════════════════════
#  w2a — Makefile
#  Go APK builder (wta) + end-to-end demo
# ═══════════════════════════════════════════════════════════

GO      ?= go
BIN_DIR ?= bin
CMDS    := $(notdir $(wildcard cmd/*))

.PHONY: all build test test-v fmt vet clean

all: build

# ─── Build ───────────────────────────────────────────────────
build:
	@mkdir -p $(BIN_DIR)
	@for c in $(CMDS); do \
		echo "==> building $$c"; \
		$(GO) build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$$c ./cmd/$$c || exit 1; \
	done

# ─── Test / lint ─────────────────────────────────────────────
test:
	$(GO) test ./...

test-v:
	$(GO) test -v ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BIN_DIR)

# ═══════════════════════════════════════════════════════════
#  Wikipedia demo — end-to-end: build → sign → install → run
# ═══════════════════════════════════════════════════════════
WIKI_DIR      ?= $(HOME)/build-demo/myapp
WIKI_TEMPLATE ?= $(HOME)/build-demo/webview_shell.apk
WIKI_OUT      ?= /sdcard/Download
WIKI_PKG      ?= com.webtoapp
WIKI_YAML     ?= wta.yaml
ANDROID_HOME  ?= $(HOME)/android-sdk
KEYSTORE      ?= release.p12
KEYSTORE_PASS ?= testpass
KEY_ALIAS     ?= myalias

.PHONY: wiki wiki-build wiki-sign wiki-install wiki-run wiki-clean

wiki: wiki-run
	@echo ""
	@echo "✅ Wikipedia demo: built, signed, installed, launched"

wiki-build: build
	@echo ""
	@echo "══ 1/4 build APK ══"
	cd $(WIKI_DIR) && \
	  WTA_TEMPLATE=$(WIKI_TEMPLATE) WTA_SKIP_MANIFEST=1 \
	  $(CURDIR)/$(BIN_DIR)/wta build -c $(WIKI_YAML) -v

wiki-sign: wiki-build
	@echo ""
	@echo "══ 2/4 zipalign + sign ══"
	@BT=$$(dirname "$$(find $(ANDROID_HOME)/build-tools -name aapt2 | head -1)"); \
	 ZIP=$$(find $(ANDROID_HOME)/build-tools -name zipalign | head -1); \
	 cd $(WIKI_DIR) && \
	 $$ZIP -f -p 4 $(WIKI_OUT)/wikipedia-reader.apk aligned.apk && \
	 $$BT/apksigner sign \
	    --ks $(KEYSTORE) --ks-type PKCS12 \
	    --ks-pass pass:$(KEYSTORE_PASS) \
	    --ks-key-alias $(KEY_ALIAS) --key-pass pass:$(KEYSTORE_PASS) \
	    --out $(WIKI_OUT)/wikipedia-final.apk \
	    aligned.apk && \
	 $$BT/apksigner verify --verbose $(WIKI_OUT)/wikipedia-final.apk | head -4

wiki-install: wiki-sign
	@echo ""
	@echo "══ 3/4 install via rish ══"
	rish -c "cp $(WIKI_OUT)/wikipedia-final.apk /data/local/tmp/"
	rish -c "pm install -d -r /data/local/tmp/wikipedia-final.apk"
	rish -c "rm /data/local/tmp/wikipedia-final.apk"

wiki-run: wiki-install
	@echo ""
	@echo "══ 4/4 launch + logcat ══"
	rish -c "logcat -c"
	rish -c "monkey -p $(WIKI_PKG) -c android.intent.category.LAUNCHER 1" >/dev/null
	@sleep 4
	@echo ""
	@echo "── Shell logs ──"
	@rish -c "logcat -d | grep -iE 'ShellModeManager|ShellScreen|WebViewManager' | tail -12"

wiki-clean:
	rm -f $(WIKI_DIR)/aligned.apk
	rm -f $(WIKI_OUT)/wikipedia-reader.apk $(WIKI_OUT)/wikipedia-final.apk

# ─── Full test suite ───
.PHONY: test-all
test-all: test
	@echo ""
	@echo "═══ wta test examples-wta/ ═══"
	WTA_TEMPLATE=$(HOME)/build-demo/webview_shell.apk \
	WTA_SKIP_MANIFEST=1 \
	$(BIN_DIR)/wta test examples-wta/

# ─── Full clean (bin + test artifacts) ───
.PHONY: clean-all
clean-all: clean
	rm -rf examples-wta/build/
	@echo "[OK] bin/ + examples-wta/build/ cleaned"


# ═══════════════════════════════════════════════════════════
# Full demo cycle: build → sign → install → launch → logcat
# ═══════════════════════════════════════════════════════════
DEMO_YAML     ?= $(HOME)/build-demo/myapp/wta.yaml
DEMO_TEMPLATE ?= $(HOME)/build-demo/webview_shell.apk
DEMO_KEYSTORE ?= $(HOME)/build-demo/myapp/release.p12
DEMO_OUT      ?= /sdcard/Download/demo

.PHONY: demo demo-sign demo-install demo-run

demo: demo-run
	@echo ""
	@echo "✅ demo installed & launched"

demo-run: demo-install
	@echo "── 4/4 launch + logcat ──"
	@rish -c "logcat -c"
	@rish -c "am start -n com.webtoapp/.ui.shell.ShellActivity"
	@sleep 5
	@rish -c "logcat -d" 2>&1 | grep -E "GENERAL" | tail -15 || true

demo-install: demo-sign
	@echo "── 3/4 install (force) ──"
	@rish -c "pm uninstall com.webtoapp" 2>/dev/null || true
	@rish -c "cp $(DEMO_OUT)-final.apk /data/local/tmp/demo.apk"
	@rish -c "pm install /data/local/tmp/demo.apk"
	@rish -c "rm /data/local/tmp/demo.apk"

demo-sign: build
	@echo "── 1/4 build ──"
	@WTA_TEMPLATE=$(DEMO_TEMPLATE) WTA_SKIP_MANIFEST=1 \
	 $(BIN_DIR)/wta build -c $(DEMO_YAML) --out $(DEMO_OUT).apk -v
	@echo ""
	@echo "── 2/4 zipalign + sign ──"
	@BT=$$(dirname "$$(find $(ANDROID_HOME)/build-tools -name aapt2 | head -1)"); \
	 ZIP=$$(find $(ANDROID_HOME)/build-tools -name zipalign | head -1); \
	 $$ZIP -f -p 4 $(DEMO_OUT).apk $(DEMO_OUT)-aligned.apk && \
	 $$BT/apksigner sign \
	   --ks $(DEMO_KEYSTORE) --ks-type PKCS12 \
	   --ks-pass pass:testpass \
	   --ks-key-alias myalias --key-pass pass:testpass \
	   --out $(DEMO_OUT)-final.apk $(DEMO_OUT)-aligned.apk

# ─── Demo without rish: build + sign + open system installer ───
.PHONY: demo-norish demo-verify

demo-norish: demo-sign
	@echo ""
	@echo "── 3/4 open system installer ──"
	@echo "   (подтверди установку на экране телефона)"
	@termux-open $(DEMO_OUT)-final.apk
	@sleep 2
	@echo ""
	@echo "── 4/4 verify (без rish) ──"
	@$(MAKE) demo-verify

# ─── Verify APK без установки ───
demo-verify:
	@echo "═══ APK: $(DEMO_OUT)-final.apk ═══"
	@ls -la $(DEMO_OUT)-final.apk 2>/dev/null || { echo "❌ нет APK"; exit 1; }
	@echo ""
	@echo "── badging ──"
	@BT=$$(dirname "$$(find $(ANDROID_HOME)/build-tools -name aapt2 | head -1)"); \
	 $$BT/aapt2 dump badging $(DEMO_OUT)-final.apk 2>&1 | grep -E "package:|application-label|launchable" | head -5
	@echo ""
	@echo "── signature ──"
	@BT=$$(dirname "$$(find $(ANDROID_HOME)/build-tools -name aapt2 | head -1)"); \
	 if $$BT/apksigner verify -v $(DEMO_OUT)-final.apk 2>&1 | grep -E "^Verifies|^Verified using"; then \
	   echo "✅ signature OK"; \
	 else \
	   echo "❌ signature FAILED"; exit 1; \
	 fi
	@echo ""
	@echo "── app_config.json ──"
	@unzip -p $(DEMO_OUT)-final.apk assets/app_config.json
	@echo ""
	@echo "── icon + arsc ──"
	@unzip -l $(DEMO_OUT)-final.apk | grep -E "res/qx.png|resources.arsc"
