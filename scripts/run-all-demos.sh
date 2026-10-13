#!/data/data/com.termux/files/usr/bin/bash
# run-all-demos.sh — full end-to-end для всех yaml в examples-wta/
# build → sign → install → launch → logcat → результат
#
# Usage:
#   scripts/run-all-demos.sh                  # все yaml
#   scripts/run-all-demos.sh --filter go      # только go-примеры
#   scripts/run-all-demos.sh --skip-build     # использовать существующие APK

set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EX_DIR="$ROOT/examples-wta"
BIN="$ROOT/bin/wta"
OUT="/sdcard/Download/wta-demos"
PKG="com.webtoapp"
KS="$HOME/build-demo/myapp/release.p12"
KS_PASS="testpass"
KEY_ALIAS="myalias"
ANDROID_HOME="${ANDROID_HOME:-$HOME/android-sdk}"
TEMPLATE="${WTA_TEMPLATE:-$HOME/build-demo/webview_shell.apk}"

SKIP_BUILD=0
FILTER=""

while [ $# -gt 0 ]; do
    case "$1" in
        --skip-build) SKIP_BUILD=1; shift ;;
        --filter)     FILTER="$2"; shift 2 ;;
        --filter=*)   FILTER="${1#--filter=}"; shift ;;
        *) echo "unknown arg: $1"; exit 2 ;;
    esac
done

mkdir -p "$OUT"

# ─── Инструменты ────────────────────────────────────
BT="$(dirname "$(find "$ANDROID_HOME/build-tools" -name aapt2 | head -1)")"
ZIP="$(find "$ANDROID_HOME/build-tools" -name zipalign | head -1)"
APKSIGNER="$BT/apksigner"

if [ ! -x "$BIN" ]; then
    echo "[!] $BIN not found — run 'make build' first"
    exit 1
fi
if [ ! -f "$KS" ]; then
    echo "[!] keystore not found: $KS"
    exit 1
fi

# ─── Список yaml ─────────────────────────────────────
mapfile -t YAMLS < <(
    for f in "$EX_DIR"/*.yaml; do
        [ -f "$f" ] || continue
        name="$(basename "$f")"
        if [ -n "$FILTER" ] && [[ "$name" != *"$FILTER"* ]]; then
            continue
        fi
        echo "$f"
    done | sort
)

if [ ${#YAMLS[@]} -eq 0 ]; then
    echo "[!] no yaml files (filter='$FILTER')"
    exit 1
fi

echo "═══════════════════════════════════════════════════════"
echo "  Full end-to-end demo run"
echo "  examples:  ${#YAMLS[@]}"
echo "  template:  $TEMPLATE"
echo "  output:    $OUT"
echo "  package:   $PKG"
echo "═══════════════════════════════════════════════════════"
echo ""

# ─── Ожидаемые appType в логе по имени yaml ──────────
declare -A EXPECT=(
    ["01-web"]="WEB"
    ["02-html"]="HTML"
    ["03-frontend"]="FRONTEND"
    ["04-go"]="GO"
    ["05-gallery"]="GALLERY"
    ["06-image"]="IMAGE"
    ["07-video"]="VIDEO"
    ["08-multiweb"]="MULTI_WEB"
    ["09-with-permissions"]="WEB"
    ["10-full-featured"]="WEB"
    ["11-sample-html"]="HTML"
    ["12-sample-go-gin"]="GO"
    ["13-sample-go-fiber"]="GO"
    ["15-wiki-article"]="WEB"
)

PASS=0; FAIL=0; SKIP=0

for Y in "${YAMLS[@]}"; do
    NAME="$(basename "$Y" .yaml)"
    BASE="${NAME%-*}"   # для 04-go → "04"
    EXPECT_TYPE="${EXPECT[$NAME]:-?}"

    APK_RAW="$OUT/${NAME}.apk"
    APK_FINAL="$OUT/${NAME}-final.apk"
    APK_ALIGNED="/sdcard/Download/${NAME}-aligned.apk"

    echo ""
    echo "═══════════════════════════════════════════════════════"
    echo "  [$NAME]  (expect appType='$EXPECT_TYPE')"
    echo "═══════════════════════════════════════════════════════"

    # ─── 1. Build ─────────────────────────────────────
    if [ "$SKIP_BUILD" -eq 1 ] && [ -f "$APK_RAW" ]; then
        echo "  [1/5] build    → SKIP (existing $APK_RAW)"
    else
        echo "  [1/5] build    → $APK_RAW"
        if ! WTA_TEMPLATE="$TEMPLATE" WTA_SKIP_MANIFEST=1 \
             "$BIN" build -c "$Y" --out "$APK_RAW" >/dev/null 2>&1; then
            echo "  ✗ BUILD FAILED"
            FAIL=$((FAIL+1))
            continue
        fi
    fi

    # ─── 2. Sign ──────────────────────────────────────
    echo "  [2/5] sign     → $APK_FINAL"
    "$ZIP" -f -p 4 "$APK_RAW" "$APK_ALIGNED" >/dev/null 2>&1
    "$APKSIGNER" sign \
        --ks "$KS" --ks-type PKCS12 \
        --ks-pass "pass:$KS_PASS" \
        --ks-key-alias "$KEY_ALIAS" --key-pass "pass:$KS_PASS" \
        --out "$APK_FINAL" "$APK_ALIGNED" >/dev/null 2>&1
    if [ ! -f "$APK_FINAL" ]; then
        echo "  ✗ SIGN FAILED"
        FAIL=$((FAIL+1))
        continue
    fi

    # ─── 3. Install ───────────────────────────────────
    echo "  [3/5] install  → rish pm install -d -r"
    rish -c "cp $APK_FINAL /data/local/tmp/${NAME}-final.apk" >/dev/null 2>&1
    INSTALL_OUT="$(rish -c "pm install -d -r /data/local/tmp/${NAME}-final.apk" 2>&1 | tail -1)"
    rish -c "rm /data/local/tmp/${NAME}-final.apk" >/dev/null 2>&1
    if ! echo "$INSTALL_OUT" | grep -qi "Success"; then
        echo "  ✗ INSTALL FAILED: $INSTALL_OUT"
        FAIL=$((FAIL+1))
        continue
    fi
    echo "         → $INSTALL_OUT"

    # ─── 4. Launch ────────────────────────────────────
    echo "  [4/5] launch   → monkey"
    rish -c "logcat -c" >/dev/null 2>&1
    rish -c "am force-stop $PKG" >/dev/null 2>&1
    sleep 0.5
    rish -c "monkey -p $PKG -c android.intent.category.LAUNCHER 1" >/dev/null 2>&1
    sleep 4

    # ─── 5. Logcat ────────────────────────────────────
    echo "  [5/5] logcat   → ShellModeManager / ShellScreen"
    LOG="$(rish -c "logcat -d | grep -iE 'ShellModeManager|ShellScreen|WebViewManager|appType|targetUrl|entryUrl|Page finished|error|Конфиг|FATAL' | tail -20" 2>&1)"

    # Извлекаем appType из лога
    APP_TYPE="$(echo "$LOG" | grep -oE "appType='[^']+'" | head -1 | sed "s/appType='//;s/'//")"
    SHELL_OK="$(echo "$LOG" | grep -c "配置有效\|Конфиг валиден" 2>/dev/null || echo 0)"
    PAGE_OK="$(echo "$LOG" | grep -c "Page finished" 2>/dev/null || echo 0)"
    FATAL="$(echo "$LOG" | grep -c "FATAL\|AndroidRuntime" 2>/dev/null || echo 0)"

    # ─── Вердикт ──────────────────────────────────────
    if [ "$FATAL" -gt 0 ]; then
        echo "  ✗ CRASH"
        echo "$LOG" | grep -iE "FATAL|Caused by" | head -3 | sed 's/^/       /'
        FAIL=$((FAIL+1))
    elif [ -z "$APP_TYPE" ]; then
        echo "  ? NO appType in log (shell didn't start?)"
        echo "$LOG" | tail -3 | sed 's/^/       /'
        FAIL=$((FAIL+1))
    else
        echo "  ✓ appType='$APP_TYPE'  shell=$SHELL_OK  page=$PAGE_OK  fatal=$FATAL"
        PASS=$((PASS+1))
    fi

    # Очистка aligned
    rm -f "$APK_ALIGNED" 2>/dev/null
done

# ─── Сводка ─────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════"
echo "  RESULT"
echo "═══════════════════════════════════════════════════════"
echo "  passed: $PASS"
echo "  failed: $FAIL"
echo "  skipped: $SKIP"
echo "  total:  ${#YAMLS[@]}"
echo ""
echo "  APKs in $OUT/"
ls -la "$OUT"/*.apk 2>/dev/null | wc -l
