#!/usr/bin/env python3
"""Membaca jawaban GitHub API check-runs dari stdin dan menilai apakah CI sebuah commit lolos.

Dipakai deploy/rilis.sh: python3 deploy/cek-ci.py <sha> <nama job> < jawaban.json
Keluar 0 hanya bila check run job itu dari GitHub Actions untuk commit itu sudah completed dan success.
Selain itu keluar 1. Satu baris pesan selalu dicetak ke stdout untuk ditampilkan rilis.sh.
"""
import json
import sys


def nilai(data, sha, nama):
    try:
        runs = data["check_runs"]
    except (TypeError, KeyError):
        return False, "Jawaban GitHub tidak berisi check_runs."
    # Hanya check run dari GitHub Actions yang dihitung: app lain yang terpasang di repo bisa membuat check run
    # dengan nama yang sama.
    cocok = [
        r for r in runs
        if isinstance(r, dict) and r.get("name") == nama and r.get("head_sha") == sha
        and (r.get("app") or {}).get("slug") == "github-actions"
    ]
    if not cocok:
        return False, f"CI job {nama} untuk commit {sha[:12]} tidak ditemukan. Tunggu CI berjalan, lalu ulangi."
    # Job yang dijalankan ulang membuat check run baru; yang berlaku adalah yang terbaru (id terbesar).
    r = max(cocok, key=lambda x: x.get("id") or 0)
    status, hasil = r.get("status"), r.get("conclusion")
    url = r.get("html_url") or ""
    if status != "completed":
        return False, f"CI job {nama} belum selesai (status {status}). Tunggu sampai selesai. {url}".rstrip()
    if hasil != "success":
        return False, f"CI job {nama} tidak lolos (hasil {hasil}). Rilis ditolak. {url}".rstrip()
    return True, f"CI job {nama} lolos untuk commit {sha[:12]}."


def main():
    if len(sys.argv) != 3:
        print("Pemakaian: cek-ci.py <sha> <nama job> < jawaban.json")
        return 1
    try:
        data = json.load(sys.stdin)
    except ValueError:
        print("Jawaban GitHub bukan JSON yang sah.")
        return 1
    ok, pesan = nilai(data, sys.argv[1], sys.argv[2])
    print(pesan)
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
