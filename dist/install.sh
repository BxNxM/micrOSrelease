#!/bin/sh

set -eu

repository="${MICROSCTL_REPOSITORY:-BxNxM/micrOSrelease}"
ref="${MICROSCTL_REF:-main}"
os="$(uname -s)"
arch="$(uname -m)"
output="microsctl"

case "$os" in
	Darwin)
		case "$arch" in
			arm64|aarch64) asset="microsctl-darwin-arm64" ;;
			*) echo "Unsupported macOS architecture: $arch" >&2; exit 1 ;;
		esac
		;;
	Linux)
		case "$arch" in
			x86_64|amd64) asset="microsctl-linux-amd64" ;;
			aarch64|arm64) asset="microsctl-linux-arm64" ;;
			*) echo "Unsupported Linux architecture: $arch" >&2; exit 1 ;;
		esac
		;;
	MINGW*|MSYS*|CYGWIN*)
		case "$arch" in
			x86_64|amd64) asset="microsctl-windows-amd64.exe" ;;
			*) echo "Unsupported Windows architecture: $arch" >&2; exit 1 ;;
		esac
		output="microsctl.exe"
		;;
	*)
		echo "Unsupported operating system: $os" >&2
		exit 1
		;;
esac

url="https://raw.githubusercontent.com/$repository/$ref/dist/$asset"

destination="$(pwd)/$output"
temporary="$destination.download.$$"
trap 'rm -f "$temporary"' EXIT HUP INT TERM

echo "Downloading $asset..."
if command -v curl >/dev/null 2>&1; then
	curl -fL --retry 3 --output "$temporary" "$url"
elif command -v wget >/dev/null 2>&1; then
	wget -O "$temporary" "$url"
else
	echo "curl or wget is required to install microsctl." >&2
	exit 1
fi

chmod +x "$temporary"
mv -f "$temporary" "$destination"
trap - EXIT HUP INT TERM

echo "Installed microsctl at $destination"
