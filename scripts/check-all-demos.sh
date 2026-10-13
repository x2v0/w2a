#!/data/data/com.termux/files/usr/bin/bash
# check-all-demos.sh — сборка + установка + проверка каждого примера.
# Генерирует временный yaml В examples-wta/ (чтобы ./source нашёлся).

set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EX_DIR="$ROOT/examples-wta"
BIN="$ROOT/bin/wta"
TMP_DIR="/sdcard/Download/w2a-demos"
ICON_PATH="$HOME/build-demo/myapp/icon-fg.png"
TEMPLATE="$HOME/build-demo/webview_shell.apk"  # ignore env override
KS="$HOME/build-demo/myapp/release.p12"
ANDROID_HOME="${ANDROID_HOME:-$HOME/android-sdk}"

mkdir -p "$TMP_DIR"
BT="$(dirname "$(find "$ANDROID_HOME/build-tools" -name aapt2 | head -1)")"
ZIP="$(find "$ANDROID_HOME/build-tools" -name zipalign | head -1)"

PASS=0; FAIL=0

cleanup() {
    rm -f "$EX_DIR"/.tmp-*.yaml
}
trap cleanup EXIT

check_one() {
    local name="$1" pkg="$2" expect="$3"

    echo ""
    echo "═══════════════════════════════════════════════════════"
    echo "  $name  (pkg=$pkg, expect=$expect)"
    echo "═══════════════════════════════════════════════════════"

    # 1. Генерируем yaml В examples-wta/ (source найдётся)
    rm -f "$TMP_DIR/$name.apk" "$TMP_DIR/$name-aligned.apk" \
          "$TMP_DIR/$name-final.apk" "$TMP_DIR/$name-final.apk.idsig"
    local tmp_yaml="$EX_DIR/.tmp-$name.yaml"
    local apk="$TMP_DIR/$name.apk"
    local final="$TMP_DIR/$name-final.apk"
    local aligned="$TMP_DIR/$name-aligned.apk"

    # Подменяем 3 поля: package (уникальный), name (короткое — для in-place
    # label patch, где newName ≤ oldName), icon (наш PNG для ReplaceIcon).
    if ! sed -E \
        -e "s|^  package:.*|  package: $pkg|" \
        -e "s|^  name:.*|  name: w2a|" \
        -e "s|^  icon:.*|  icon: $ICON_PATH|" \
        "$EX_DIR/$name.yaml" > "$tmp_yaml"; then
        echo "  ✗ sed failed"
        FAIL=$((FAIL+1)); return
    fi

    # Если в yaml не было поля icon: — добавляем после versionName.
    if ! grep -q "^  icon:" "$tmp_yaml"; then
        sed -i "0,/^  versionName:.*/s|^  versionName:.*|&\n  icon: $ICON_PATH|" "$tmp_yaml"
    fi

    # 2. Build
    if ! WTA_TEMPLATE="$TEMPLATE" WTA_SKIP_MANIFEST=1 \
         "$BIN" build -c "$tmp_yaml" --out "$apk" >/dev/null 2>&1; then
        echo "  ✗ BUILD FAILED"
        # Показать причину
        WTA_TEMPLATE="$TEMPLATE" WTA_SKIP_MANIFEST=1 \
            "$BIN" build -c "$tmp_yaml" --out "$apk" 2>&1 | tail -2 | sed 's/^/    /'
        FAIL=$((FAIL+1)); return
    fi

    # 3. Sign
    "$ZIP" -f -p 4 "$apk" "$aligned" >/dev/null 2>&1
    "$BT/apksigner" sign \
        --ks "$KS" --ks-type PKCS12 \
        --ks-pass pass:testpass --ks-key-alias myalias --key-pass pass:testpass \
        --out "$final" "$aligned" >/dev/null 2>&1

    # 4. Install
    rish -c "cp $final /data/local/tmp/$name.apk" >/dev/null 2>&1
    out=$(rish -c "pm install /data/local/tmp/$name.apk" 2>&1)
    rish -c "rm /data/local/tmp/$name.apk" >/dev/null 2>&1

    # Показать что вернул install (для диагностики)
    if ! echo "$out" | grep -qi "success"; then
        echo "  ✗ INSTALL: $(echo "$out" | tr '\n' ' ' | head -c 200)"
        FAIL=$((FAIL+1)); return
    fi

    # 5. Launch + logcat
    rish -c "logcat -c" >/dev/null 2>&1
    rish -c "am start -n $pkg/com.webtoapp.ui.shell.ShellActivity" >/dev/null 2>&1
    sleep 4
    log=$(rish -c "logcat -d" 2>&1 | grep "GENERAL" | tail -20)

    app_type=$(echo "$log" | grep -oE "appType='[^']+'" | head -1 | sed "s/appType='//;s/'//")
    page_ok=$(echo "$log" | awk '/Page finished/{c++} END{print c+0}')
    fatal=$(echo "$log" | awk '/FATAL|AndroidRuntime/{c++} END{print c+0}')

    # 6. Uninstall
    rish -c "pm uninstall $pkg" >/dev/null 2>&1

    if [ "$fatal" -gt 0 ]; then
        echo "  ✗ CRASH"
        echo "$log" | grep -E "FATAL|Caused by" | head -3 | sed 's/^/    /'
        FAIL=$((FAIL+1))
    elif [ -z "$app_type" ]; then
        echo "  ? NO appType in log"
        echo "$log" | tail -3 | sed 's/^/    /'
        FAIL=$((FAIL+1))
    else
        echo "  ✓ appType='$app_type'  page=$page_ok  fatal=$fatal"
        PASS=$((PASS+1))
    fi
}

check_one "01-web"               "com.w2a.web"     "WEB"
check_one "02-html"              "com.w2a.html"    "HTML"
check_one "03-frontend"          "com.w2a.spa"     "FRONTEND"
check_one "04-go"                "com.w2a.go"      "GO_APP"
check_one "05-gallery"           "com.w2a.gal" "GALLERY"
check_one "06-image"             "com.w2a.img"   "IMAGE"
check_one "07-video"             "com.w2a.vid"   "VIDEO"
check_one "08-multiweb"          "com.w2a.mw"   "MULTI_WEB"
check_one "09-with-permissions"  "com.w2a.perm"    "WEB"
check_one "10-full-featured"     "com.w2a.full"    "WEB"
check_one "11-sample-html"       "com.w2a.sh"  "HTML"
check_one "12-sample-go-gin"     "com.w2a.gin"     "GO_APP"
check_one "13-sample-go-fiber"   "com.w2a.fb"   "GO_APP"
check_one "15-wiki-article"      "com.w2a.wiki"    "WEB"

echo ""
echo "═══════════════════════════════════════════════════════"
echo "  RESULT: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════════════════════════"
