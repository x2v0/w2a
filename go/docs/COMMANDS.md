# Command reference

## wta-checkdrift

```bash
wta-checkdrift                                  # defaults
wta-checkdrift -payload file.kt -shell file.kt
```

Output: `check_config_field_drift: OK (payload=458 shell=487 ...)`

## wta-muslbridge

```bash
wta-muslbridge -ndk "$ANDROID_NDK_HOME"
wta-muslbridge -abis arm64-v8a                  # one ABI only
wta-muslbridge -tarball /path/musl-1.2.5.tar.gz # skip download
```

## wta-validatemodules

```bash
wta-validatemodules -repo .
echo $?                                          # 0 OK, 1 errors
```

## wta-gensubmissions

```bash
wta-gensubmissions -repo-root . --offline
wta-gensubmissions --owner shiaho777 --repo web-to-app --token $GITHUB_TOKEN
wta-gensubmissions -repo-root . --check          # CI guard
```

## wta-clonehost

```bash
wta-clonehost -aar clone-host/build/outputs/aar/clone-host-release.aar \
  -sdk "$ANDROID_HOME" -out app/src/main/assets/clone_host
```

## wta-stackbundle

```bash
wta-stackbundle -src "feature-stacks/*/build/outputs/stack/*.zip" \
  -dst app/src/main/assets/stacks
```

## wta-shelltemplate

```bash
wta-shelltemplate -src shell/build/outputs/apk/release/shell-release.apk \
  -dst app/src/main/assets/template/webview_shell.apk
```

## wta-buildapk

### Inspect
```bash
wta-buildapk -template app.apk -inspect [-v]
wta-buildapk -template app.apk -read-package
wta-buildapk -template app.apk -list-permissions
wta-buildapk -template app.apk -list-assets
wta-buildapk -template app.apk -list-strings -strings-offset 0 -strings-limit 50
wta-buildapk -template app.apk -list-locales
```

### Manifest
```bash
wta-buildapk -template app.apk -package com.new.app
wta-buildapk -template app.apk -version-code 42 -version-name "2.0"
wta-buildapk -template app.apk -set-app-name "My App"
wta-buildapk -template app.apk -add-permission "android.permission.CAMERA"
wta-buildapk -template app.apk -remove-permission android.permission.QUERY_ALL_PACKAGES
```

### Resources
```bash
wta-buildapk -template app.apk -set-string 5="Replaced"
wta-buildapk -template app.apk -strip-locale ar
wta-buildapk -template app.apk -strip-locales ar,ru,zh
wta-buildapk -template app.apk -keep-locales en
wta-buildapk -template app.apk -strip-script cyrillic -script-replacement ""
wta-buildapk -template app.apk -normalize-locale
wta-buildapk -template app.apk -replace-icon icon.png
```

Scripts supported: `arabic`, `cyrillic`, `hebrew`, `greek`, `devanagari`,
`thai`, `cjk`, `kana`, `hangul`.

### Files
```bash
wta-buildapk -template app.apk -add-asset local.txt=assets/foo.txt
wta-buildapk -template app.apk -add-so libfoo.so=lib/arm64-v8a/libfoo.so
```

### Build
```bash
wta-buildapk -template shell.apk \
  -url https://example.com \
  -app-name "My App" \
  -package com.example.app \
  -version-code 1 -version-name "1.0" \
  -icon icon.png \
  -out myapp.apk
```

## wta-signapk

```bash
wta-signapk -in myapp.apk -out signed.apk -keystore release.pem
wta-signapk -in myapp.apk -out signed.apk -keystore release.pem -v1=true
```

Keystore format: PEM with `RSA PRIVATE KEY` + `CERTIFICATE` blocks.
Generate test key:
```bash
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=Test/O=WebToApp/C=US"
cat key.pem cert.pem > release.pem
```
