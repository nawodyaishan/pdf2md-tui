#!/usr/bin/env bash
set -euo pipefail

tag="${1:?usage: update-homebrew-formula.sh <tag>}"

if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid tag: $tag" >&2
  exit 1
fi

if [[ -z "${HOMEBREW_TAP_TOKEN:-}" ]]; then
  echo "HOMEBREW_TAP_TOKEN is required" >&2
  exit 1
fi

version="${tag#v}"
repo="nawodyaishan/pdf2md-tui"
tap_repo="nawodyaishan/homebrew-tap"
source_url="https://github.com/${repo}/archive/refs/tags/${tag}.tar.gz"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

archive="${tmp_dir}/${tag}.tar.gz"
curl -fsSL "$source_url" -o "$archive"
sha256="$(shasum -a 256 "$archive" | awk '{print $1}')"

tap_dir="${tmp_dir}/homebrew-tap"
git clone --depth 1 "https://x-access-token:${HOMEBREW_TAP_TOKEN}@github.com/${tap_repo}.git" "$tap_dir"

mkdir -p "${tap_dir}/Formula"
cat > "${tap_dir}/Formula/pdf2md-tui.rb" <<FORMULA
# typed: false
# frozen_string_literal: true

class Pdf2mdTui < Formula
  desc "High-performance TUI tool for batch PDF to LLM-friendly Markdown conversion"
  homepage "https://github.com/${repo}"
  url "${source_url}"
  sha256 "${sha256}"
  license "MIT"
  version "${version}"

  depends_on "go" => :build

  def install
    build_date = Time.now.utc.strftime("%Y-%m-%dT%H:%M:%SZ")
    go_version = Utils.safe_popen_read("go", "version").split[2]
    ldflags = %W[
      -s -w
      -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Version=#{version}
      -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Commit=#{version}
      -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Date=#{build_date}
      -X github.com/nawodyaishan/pdf2md-tui/pkg/version.GoVersion=#{go_version}
    ]

    system "go", "build", *std_go_args(ldflags: ldflags.join(" ")), "./cmd/pdf2md-tui"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/pdf2md-tui version")
  end
end
FORMULA

if [[ -f "${tap_dir}/Casks/pdf2md-tui.rb" ]]; then
  git -C "$tap_dir" rm -f Casks/pdf2md-tui.rb
fi

git -C "$tap_dir" add Formula/pdf2md-tui.rb

if git -C "$tap_dir" diff --cached --quiet; then
  echo "Homebrew Formula already up to date for ${tag}"
  exit 0
fi

git -C "$tap_dir" config user.name "github-actions[bot]"
git -C "$tap_dir" config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git -C "$tap_dir" commit -m "Brew formula update for pdf2md-tui ${tag}"
git -C "$tap_dir" push "https://x-access-token:${HOMEBREW_TAP_TOKEN}@github.com/${tap_repo}.git" HEAD:main
