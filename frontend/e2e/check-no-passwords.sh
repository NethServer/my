#!/usr/bin/env bash
#  Copyright (C) 2026 Nethesis S.r.l.
#  SPDX-License-Identifier: GPL-3.0-or-later
#
# Fails when a credential the suite signs in with is anywhere in its output:
# the HTML report (including the data zip embedded in index.html), the
# per-test results, and the traces inside both. CI runs it before uploading
# those as an artifact, and uploads nothing when it fails — the repository is
# public, so an artifact is readable by anyone signed in to GitHub.
#
#   bash e2e/check-no-passwords.sh [dir...]   # from frontend/; default:
#                                             # playwright-report test-results
#
# The credentials it looks for, from whichever of these is present:
#
#   E2E_SMOKE_PASSWORD           the smoke account (e2e-smoke.yml)
#   backend/.api-registry.json   the owner and every persona (e2e-main.yml)
#
# A spec that starts using another password, secret or API key must add it
# here, and its workflow must pass it to this step's env — a credential this
# script is not told about is one it cannot find. Matches are reported by file
# name only, never by value.

set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
registry="$here/../../backend/.api-registry.json"

if [ "$#" -eq 0 ]; then
  set -- playwright-report test-results
fi

scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT

raw="$scratch/raw"
: >"$raw"
if [ -n "${E2E_SMOKE_PASSWORD:-}" ]; then
  printf '%s\n' "$E2E_SMOKE_PASSWORD" >>"$raw"
fi
if [ -f "$registry" ]; then
  jq -r '.owner.password // empty, (.users // {} | .[].password // empty)' "$registry" >>"$raw"
fi

# Traces and report data are JSON, where a quote or a backslash in a password
# is escaped: look for that spelling too.
patterns="$scratch/patterns"
{
  cat "$raw"
  jq -Rr '@json | .[1:-1]' <"$raw"
} | grep -v '^$' | sort -u >"$patterns" || true

if [ ! -s "$patterns" ]; then
  echo "No credentials to look for (E2E_SMOKE_PASSWORD unset, no registry): nothing to check."
  exit 0
fi

# Unpacked under their own path, so a match inside one names the archive.
unpacked="$scratch/unpacked"
for dir in "$@"; do
  [ -d "$dir" ] || continue

  # An archive that cannot be read cannot be vouched for: fail rather than
  # skip it.
  while IFS= read -r -d '' zip; do
    mkdir -p "$unpacked/$zip"
    if ! unzip -qo "$zip" -d "$unpacked/$zip"; then
      echo "::error::cannot unpack $zip to check it"
      exit 1
    fi
  done < <(find "$dir" -type f -name '*.zip' -print0)

  if [ -f "$dir/index.html" ]; then
    embedded=$(sed -n 's/.*<template id="playwrightReportBase64">data:application\/zip;base64,\([^<]*\)<\/template>.*/\1/p' "$dir/index.html")
    if [ -n "$embedded" ]; then
      mkdir -p "$unpacked/$dir"
      printf '%s' "$embedded" | base64 -d >"$unpacked/$dir/embedded.zip"
      unzip -qo "$unpacked/$dir/embedded.zip" -d "$unpacked/$dir/index.html"
    fi
  fi
done

mkdir -p "$unpacked"
leaks=$(grep -rlF -f "$patterns" "$@" "$unpacked" 2>/dev/null || true)
if [ -n "$leaks" ]; then
  echo "::error::a credential appears in the test output, which must not be uploaded:"
  printf '%s\n' "$leaks" | sed "s|^$unpacked/|inside |"
  exit 1
fi

echo "No credentials in: $*"
