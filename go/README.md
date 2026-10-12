# wta-go

Go tooling for **WebToApp** — the Android app that builds APKs on-device.

This module replaces all Python CI scripts and Gradle helper tasks with
statically-linked Go binaries. It also includes a minimal APK builder,
resources.arsc parser/writer, and APK signer.

## Quick start

```bash
cd go
make build
make test
./bin/wta-buildapk -h
```

Requires **Go 1.22+**. No Python, no JDK, no Gradle needed.

## Binaries

| Binary | Purpose |
|---|---|
| wta-checkdrift | Verify @SerializedName parity |
| wta-muslbridge | Build patched musl linker |
| wta-validatemodules | Validate modules/ catalog |
| wta-gensubmissions | Generate submissions.json |
| wta-clonehost | AAR to DEX |
| wta-stackbundle | Copy feature-stack ZIPs |
| wta-shelltemplate | Copy shell APK into assets |
| wta-buildapk | Build/mutate/sign APK |
| wta-signapk | Pure-Go APK signer (V1) |

## Packages

```
internal/
  apk        AXML, ZIP, manifest, permissions, assets, native libs
  apksig     APK signing (V1 JAR working, V2 WIP)
  apkutil    Android SDK build-tools detection
  arsc       resources.arsc reader/writer/merge + locales
  assetsync  File copying to app/src/main/assets
  kotlin     Kotlin source parsing
  modules    Plugin Market catalog
  musl       musl libc build with pure-Go patch applier
```

## wta-buildapk cheat sheet

### Inspect

```bash
wta-buildapk -template app.apk -inspect
wta-buildapk -template app.apk -inspect -v
wta-buildapk -template app.apk -read-package
wta-buildapk -template app.apk -list-permissions
wta-buildapk -template app.apk -list-assets
wta-buildapk -template app.apk -list-strings -strings-limit 20
wta-buildapk -template app.apk -list-locales
```

### Manifest edits

```bash
wta-buildapk -template app.apk -package com.new.app -out out.apk
wta-buildapk -template app.apk -version-code 42 -version-name "2.0"
wta-buildapk -template app.apk -set-app-name "My App"
wta-buildapk -template app.apk -add-permission "android.permission.CAMERA"
wta-buildapk -template app.apk -remove-permission android.permission.QUERY_ALL_PACKAGES
```

### Resources

```bash
wta-buildapk -template app.apk -set-string 5="Replaced"
wta-buildapk -template app.apk -strip-locale ar
wta-buildapk -template app.apk -strip-locales ar,ru
wta-buildapk -template app.apk -keep-locales en
wta-buildapk -template app.apk -strip-script cyrillic -script-replacement ""
wta-buildapk -template app.apk -normalize-locale
wta-buildapk -template app.apk -replace-icon icon.png
```

### Files

```bash
wta-buildapk -template app.apk -add-asset local.txt=assets/foo.txt
wta-buildapk -template app.apk -add-so libfoo.so=lib/arm64-v8a/libfoo.so
```

### Build and sign

```bash
wta-buildapk -template shell.apk -url https://example.com \
  -app-name "My App" -package com.example.app -icon icon.png -out myapp.apk

wta-signapk -in myapp.apk -out signed.apk -keystore release.pem
```

## Signing notes

- **V1 (JAR signing)**: fully implemented, verified by apksigner.
- **V2 (APK Signature Scheme v2)**: WIP. Use apksigner for targetSdk >= 30.

## Environment variables

| Var | Used by | Purpose |
|---|---|---|
| ANDROID_HOME | toolchain detection | locate aapt2, d8, apksigner, zipalign |
| ANDROID_NDK_HOME | wta-muslbridge | locate NDK clang |
| WTA_TEST_APK | tests | path to any APK with resources.arsc |
| GITHUB_TOKEN | wta-gensubmissions | GitHub API access |

## Testing

```bash
cd go && go test ./...
export WTA_TEST_APK=/path/to/app.apk
go test ./internal/apk/ -v
go test ./internal/arsc/ -v
```

## Known limitations

- V2 signing not verified — use apksigner for targetSdk >= 30
- ARSC merge: append-only
- ZIP64 not supported (< 4 GB APKs)
- Native libs not aligned to 16 KB
- Feature stacks not injected from Go

## Roadmap

- [ ] V2 APK signing
- [ ] V3 signing (key rotation)
- [ ] ARSC: modify complex entries
- [ ] ARSC: merge two APKs
- [ ] Incremental build cache
- [ ] Feature-stack injection from Go
- [ ] 16 KB alignment for native libs
