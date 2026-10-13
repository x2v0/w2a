#!/usr/bin/env python3
"""fix-fixtures.py — идемпотентно создать фикстуры для examples-wta.

Создаёт минимальные source-проекты, на которые ссылаются yaml-примеры:
  02-html       → html-project/index.html
  03-frontend   → dist/index.html, dist/app.js
  04-go         → goproject/main.go, goproject/go.mod
  05-gallery    → photos/01.png, photos/02.png, photos/03.png
  06-image      → photo.jpg
  07-video      → video.mp4

Повторный запуск ничего не ломает — существующие файлы не трогаются.
"""
import sys
from pathlib import Path

ROOT = Path(__file__).parent / "examples-wta"

FILES = {
    # 02-html
    "html-project/index.html": """<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Test HTML</title></head>
<body>
<h1>HTML works</h1>
<p>Loaded from assets/webroot/</p>
</body>
</html>
""",

    # 03-frontend
    "dist/index.html": """<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Test SPA</title></head>
<body>
<h1>SPA works</h1>
</body>
</html>
""",
    "dist/app.js": "console.log('[spa] loaded');\n",

    # 04-go
    "goproject/main.go": """package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<h1>Go server works</h1>")
	})
	http.ListenAndServe(":8080", nil)
}
""",
    "goproject/go.mod": "module goproject\n\ngo 1.22\n",

    # 07-video — минимальный mp4 (ftypisom box)
    "video.mp4": None,  # спец. обработка — binary
}

# 1x1 PNG (base64-декод в bytes)
PNG_1X1 = bytes([
    0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
    0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
    0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
    0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
    0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
    0x54, 0x78, 0x9C, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
    0x00, 0x00, 0x03, 0x00, 0x01,
])

# Минимальный MP4 (только ftyp box)
MP4_MINIMAL = bytes([
    0x00, 0x00, 0x00, 0x20,  # box size = 32
    0x66, 0x74, 0x79, 0x70,  # 'ftyp'
    0x69, 0x73, 0x6F, 0x6D,  # 'isom'
    0x00, 0x00, 0x02, 0x00,  # minor version
    0x69, 0x73, 0x6F, 0x6D,  # 'isom'
    0x69, 0x73, 0x6F, 0x32,  # 'iso2'
    0x61, 0x76, 0x63, 0x31,  # 'avc1'
    0x6D, 0x70, 0x34, 0x31,  # 'mp41'
])


def write_text(path: Path, content: str) -> bool:
    if path.exists():
        print(f"[i] {path.relative_to(ROOT)}: уже есть")
        return False
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    print(f"[OK] {path.relative_to(ROOT)}: создан ({len(content)}b)")
    return True


def write_bytes(path: Path, content: bytes) -> bool:
    if path.exists():
        print(f"[i] {path.relative_to(ROOT)}: уже есть")
        return False
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(content)
    print(f"[OK] {path.relative_to(ROOT)}: создан ({len(content)}b)")
    return True


def main():
    if not ROOT.is_dir():
        print(f"[!] не найдено: {ROOT}")
        sys.exit(1)

    # Text-файлы
    for rel, content in FILES.items():
        if content is None:
            continue
        write_text(ROOT / rel, content)

    # PNG (3 копии)
    for name in ("01", "02", "03"):
        write_bytes(ROOT / f"photos/{name}.png", PNG_1X1)

    # photo.jpg (используем PNG-данные — тест на формат)
    write_bytes(ROOT / "photo.jpg", PNG_1X1)

    # video.mp4
    write_bytes(ROOT / "video.mp4", MP4_MINIMAL)

    print("\n[OK] все фикстуры созданы")


if __name__ == "__main__":
    main()
