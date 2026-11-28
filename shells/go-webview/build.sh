#!/data/data/com.termux/files/usr/bin/bash
set -e
cd "$(dirname "$0")"

# SDK discovery
SDK="${ANDROID_HOME:-$HOME/android-sdk}"
# Pick the NEWEST build-tools just for d8 (Java, arch-independent).
# aapt2/zipalign/apksigner come from Termux ($PREFIX/bin, arm64) —
# SDK build-tools binaries are often x86 and won't run on Termux.
BT=$(ls -1 "$SDK/build-tools" 2>/dev/null | sort -V | tail -1)
[ -z "$BT" ] && { echo "no build-tools under $SDK/build-tools"; exit 1; }
BT="$SDK/build-tools/$BT"

AAPT2="$PREFIX/bin/aapt2"
ZIPALIGN="$PREFIX/bin/zipalign"
APKSIGNER="$PREFIX/bin/apksigner"
[ -x "$AAPT2" ]    || { echo "aapt2 missing in $PREFIX/bin (pkg install aapt2)"; exit 1; }
[ -x "$ZIPALIGN" ] || { echo "zipalign missing in $PREFIX/bin (pkg install zipalign)"; exit 1; }
[ -x "$APKSIGNER" ]|| { echo "apksigner missing in $PREFIX/bin (pkg install apksigner)"; exit 1; }
ANDROID_JAR="$SDK/platforms/android-34/android.jar"
[ ! -f "$ANDROID_JAR" ] && ANDROID_JAR=$(find "$SDK/platforms" -name android.jar | sort -V | tail -1)

echo "SDK:        $SDK"
echo "BT:         $BT"
echo "android.jar: $ANDROID_JAR"

# Keystore
# Credentials: env → sign.yaml → defaults
KS="${WTA_KEYSTORE:-$HOME/.config/w2a/keystore.p12}"

if [ -f "$HOME/.config/w2a/sign.yaml" ]; then
    eval "$(python3 -c "
import yaml, shlex
with open('$HOME/.config/w2a/sign.yaml') as f: d = yaml.safe_load(f) or {}
for k, v in [('YAML_KS', d.get('keystore', '') or ''),
             ('YAML_KSPASS', d.get('ksPass', '') or ''),
             ('YAML_ALIAS', d.get('keyAlias', '') or ''),
             ('YAML_KEYPASS', d.get('keyPass', '') or '')]:
    print(k + '=' + shlex.quote(str(v)))
")"
fi

KS="${WTA_KEYSTORE:-${YAML_KS:-$HOME/.config/w2a/keystore.p12}}"
KS_PASS="${WTA_KS_PASS:-${YAML_KSPASS:-pass:testpass}}"
KEY_ALIAS="${WTA_KEY_ALIAS:-${YAML_ALIAS:-myalias}}"
KEY_PASS="${WTA_KEY_PASS:-${YAML_KEYPASS:-pass:testpass}}"

[ ! -f "$KS" ] && { echo "keystore missing: $KS (run: wta sign --generate --cn X)"; exit 1; }

echo "KS:         $KS"
echo "ALIAS:      $KEY_ALIAS"

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
echo "WORK:       $WORK"

# 1) compile resources
"$AAPT2" compile --dir res -o "$WORK/resources.zip"
echo "[1/7] aapt2 compile OK"

# 2) link
"$AAPT2" link \
    -o "$WORK/base.apk" \
    --manifest AndroidManifest.xml \
    -I "$ANDROID_JAR" \
    -R "$WORK/resources.zip" \
    --auto-add-overlay \
    --min-sdk-version 24 \
    --target-sdk-version 29
echo "[2/7] aapt2 link OK"

# 3) javac
mkdir -p "$WORK/classes"
find src -name '*.java' > "$WORK/sources.txt"
javac -source 8 -target 8 -cp "$ANDROID_JAR" \
    -d "$WORK/classes" @"$WORK/sources.txt" 2>&1 | tail -5
echo "[3/7] javac OK"

# 4) d8
mkdir -p "$WORK/dex"
D8="$BT/d8"
"$D8" --release --min-api 24 --lib "$ANDROID_JAR" \
    --output "$WORK/dex" \
    $(find "$WORK/classes" -name '*.class')
echo "[4/7] d8 OK ($(ls -la $WORK/dex/classes.dex | awk '{print $5}') bytes)"

# 5) package APK
cp "$WORK/base.apk" "$WORK/full.apk"
( cd "$WORK/dex" && zip -q "$WORK/full.apk" classes.dex )
echo "[5/7] classes.dex added ($(ls -la "$WORK/dex/classes.dex" | awk '{print $5}') bytes)"
if [ -d "lib" ]; then
    ( cd . && zip -q -r "$WORK/full.apk" lib )
    echo "[5b] lib/ added ($(du -sh lib | cut -f1))"
else
    echo "[5b] lib/ skipped"
fi

# 6) zipalign
"$ZIPALIGN" -f 4 "$WORK/full.apk" "$WORK/aligned.apk"
echo "[6/7] zipalign OK"

# 7) sign
"$APKSIGNER" sign \
    --ks "$KS" --ks-type PKCS12 \
    --ks-pass "$KS_PASS" \
    --ks-key-alias "$KEY_ALIAS" \
    --key-pass "$KEY_PASS" \
    --out shell_webview.apk \
    "$WORK/aligned.apk"
echo "[7/7] signed"

ls -la shell_webview.apk
echo ""
echo "═══ shell_webview.apk built ═══"

