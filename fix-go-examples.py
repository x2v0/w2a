#!/usr/bin/env python3
"""fix-go-examples.py — фикс go-примеров для сборки в Termux.

1. Убирает x86_64 из abis (android/amd64 требует cgo, недоступен).
2. Дёргает 'go get' в sample-projects для заполнения go.sum
   (gin, fiber — внешние deps).
"""
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).parent

# ─── 1. abis в yaml ──────────────────────────────────────
def fix_abis():
    fixed = 0
    for y in sorted(ROOT.glob("examples-wta/*go*.yaml")):
        s = y.read_text(encoding="utf-8")
        orig = s
        s = s.replace("abis: [arm64-v8a, x86_64]", "abis: [arm64-v8a]")
        s = s.replace("abis: [x86_64, arm64-v8a]", "abis: [arm64-v8a]")
        s = s.replace("abis: [arm64-v8a,x86_64]", "abis: [arm64-v8a]")
        if s != orig:
            y.write_text(s, encoding="utf-8")
            print(f"[OK] {y.name}: abis → [arm64-v8a]")
            fixed += 1
    if fixed == 0:
        print("[i] abis: нечего править")

# ─── 2. go.sum в sample-projects ─────────────────────────
def fix_go_deps():
    projects = [
        ("go-gin-en",   "github.com/gin-gonic/gin@v1.9.1"),
        ("go-fiber-en", "github.com/gofiber/fiber/v2@v2.52.0"),
        ("go-echo-en",  ""),  # только stdlib
    ]
    base = ROOT / "app" / "src" / "main" / "assets" / "sample_projects"

    for name, dep in projects:
        d = base / name
        if not d.exists():
            print(f"[!] {name}: not found at {d}")
            continue

        gomod = d / "go.mod"
        if not gomod.exists():
            print(f"[!] {name}: no go.mod")
            continue

        gosum = d / "go.sum"
        if gosum.exists() and gosum.stat().st_size > 100:
            print(f"[i] {name}: go.sum уже есть ({gosum.stat().st_size}b)")
            continue

        if not dep:
            print(f"[i] {name}: stdlib-only, skip")
            continue

        print(f"[→] {name}: go get {dep}")
        env = {**os.environ, "GOOS": "android", "GOARCH": "arm64"}
        r = subprocess.run(
            ["go", "get", dep], cwd=d, env=env,
            capture_output=True, text=True,
        )
        if r.returncode != 0:
            print(f"[!] {name}: go get failed")
            print(r.stderr[-300:])
        else:
            size = gosum.stat().st_size if gosum.exists() else 0
            print(f"[OK] {name}: go.sum = {size}b")

def main():
    print("═══ 1. abis в yaml ═══")
    fix_abis()
    print()
    print("═══ 2. go.sum в sample-projects ═══")
    fix_go_deps()
    print()
    print("[OK] fix-go-examples.py завершён")

if __name__ == "__main__":
    main()
