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
