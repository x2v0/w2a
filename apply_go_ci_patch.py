#!/usr/bin/env python3
"""Patch Android CI + Gradle to use Go binaries."""
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
GRADLE = REPO / "app/build.gradle.kts"
CI     = REPO / ".github/workflows/android-ci.yml"

def backup(p):
    b = Path(str(p) + ".gopatch")
    shutil.copy2(p, b)
    print(f"  backup: {b.name}")

def patch_gradle():
    src = GRADLE.read_text(encoding="utf-8")
    if "WTA_MUSL_PREBUILT" in src:
        print("  [skip] gradle already patched"); return False
    anchor = 'val muslBridgeTask = tasks.register<BuildMuslBridgeTask>("buildMuslBridge") {'
    if anchor not in src:
        print(f"  [MISS] anchor not found"); return False
    idx = src.find(anchor) + len(anchor)
    insert = '\n    // Skip Gradle build when a prebuilt is provided by ./go/bin/wta-muslbridge\n    enabled = System.getenv("WTA_MUSL_PREBUILT") != "1"'
    src = src[:idx] + insert + src[idx:]
    backup(GRADLE)
    GRADLE.write_text(src, encoding="utf-8")
    print("  [ok] gradle: buildMuslBridge gated by WTA_MUSL_PREBUILT")
    return True

def patch_ci():
    src = CI.read_text(encoding="utf-8")
    if "wta-muslbridge" in src or "wta-checkdrift" in src:
        print("  [skip] CI already patched"); return False

    # 1. Replace Gradle task with Go binary call
    old_task = "            :shell:compileDebugKotlin \\\n            :app:checkConfigFieldDrift"
    new_task = "            :shell:compileDebugKotlin\n\n          ./go/bin/wta-checkdrift"
    if old_task not in src:
        print(f"  [MISS] checkConfigFieldDrift anchor not found"); return False
    src = src.replace(old_task, new_task, 1)

    # 2. Add setup-go before FIRST "Prepare Android tools"
    setup_go = (
        "      - name: Prepare Go\n"
        "        uses: actions/setup-go@v5\n"
        "        with:\n"
        "          go-version: \"1.22\"\n"
        "          cache-dependency-path: go/go.sum\n\n"
        "      - name: Build Go tools\n"
        "        working-directory: go\n"
        "        run: make build\n\n"
    )
    needle = "      - name: Prepare Android tools\n"
    pos = src.find(needle)
    if pos < 0:
        print("  [MISS] first 'Prepare Android tools' not found"); return False
    src = src[:pos] + setup_go + src[pos:]

    # 3. Add muslbridge build before "Build standard debug APK"
    bridge_step = (
        "      - name: Build musl-bridge (Go)\n"
        "        working-directory: go\n"
        "        run: |\n"
        "          ./bin/wta-muslbridge \\\n"
        "            -ndk \"$ANDROID_NDK_HOME\" \\\n"
        "            -work ../app/build/musl-bridge \\\n"
        "            -out ../app/build/generated/jniLibs/muslBridge \\\n"
        "            -patch ../scripts/patches/musl-1.2.5-wta-exec-bridge.patch \\\n"
        "            -shim ../scripts/musl-bridge/wta_mulxc3.c\n\n"
    )
    needle2 = "      - name: Build standard debug APK\n"
    pos2 = src.find(needle2)
    if pos2 < 0:
        print("  [MISS] 'Build standard debug APK' not found"); return False
    src = src[:pos2] + bridge_step + src[pos2:]

    # 4. Add WTA_MUSL_PREBUILT=1 env to all four Gradle build steps
    for name in ("Build standard debug APK", "Build standard release APK",
                 "Build gplay debug APK", "Build gplay release APK"):
        old = f"      - name: {name}\n        run: |\n"
        new = f"      - name: {name}\n        env:\n          WTA_MUSL_PREBUILT: \"1\"\n        run: |\n"
        if old in src:
            src = src.replace(old, new, 1)

    backup(CI)
    CI.write_text(src, encoding="utf-8")
    print("  [ok] ci: Go setup + muslbridge + WTA_MUSL_PREBUILT")
    return True

def main():
    print("== patch gradle ==")
    g = patch_gradle()
    print("== patch ci ==")
    c = patch_ci()
    print("\nChanges:", ("gradle" if g else "") + (" ci" if c else "") or "none")
    if g or c:
        print("\nReview: git diff app/build.gradle.kts .github/workflows/android-ci.yml")
    return 0

if __name__ == "__main__":
    sys.exit(main())
