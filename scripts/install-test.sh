#!/bin/sh

# Exercise installer platform selection without network requests or hardware.
set -eu

installer="$(CDPATH= cd -- "$(dirname -- "$0")/../dist" && pwd)/install.sh"
temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT HUP INT TERM
mkdir -p "$temporary/bin" "$temporary/output"

cat > "$temporary/bin/uname" <<'EOF'
#!/bin/sh
case "$1" in
    -s) echo "$TEST_OS" ;;
    -m) echo "$TEST_ARCH" ;;
    *) exit 1 ;;
esac
EOF

cat > "$temporary/bin/curl" <<'EOF'
#!/bin/sh
set -eu
[ "$1" = -fL ] && [ "$2" = --retry ] && [ "$3" = 3 ] && [ "$4" = --output ]
[ "$6" = "https://raw.githubusercontent.com/test/repository/test-ref/dist/$TEST_ASSET" ]
printf 'test binary\n' > "$5"
EOF

chmod +x "$temporary/bin/uname" "$temporary/bin/curl"
export PATH="$temporary/bin:$PATH"
export MICROSCTL_REPOSITORY=test/repository MICROSCTL_REF=test-ref
cd "$temporary/output"

while read -r TEST_OS TEST_ARCH TEST_ASSET output; do
    export TEST_OS TEST_ARCH TEST_ASSET
    sh "$installer" > "$temporary/log" 2>&1 || { cat "$temporary/log"; exit 1; }
    [ -x "$output" ]
    [ "$(cat "$output")" = 'test binary' ]
    rm "$output"
done <<'EOF'
Linux aarch64 microsctl-linux-arm64 microsctl
Linux arm64 microsctl-linux-arm64 microsctl
Linux x86_64 microsctl-linux-amd64 microsctl
Linux amd64 microsctl-linux-amd64 microsctl
Darwin arm64 microsctl-darwin-arm64 microsctl
MINGW64_NT x86_64 microsctl-windows-amd64.exe microsctl.exe
EOF

export TEST_OS=Linux TEST_ARCH=armv7l
if sh "$installer" > "$temporary/log" 2>&1; then
    echo 'Expected 32-bit ARM to be rejected.' >&2
    exit 1
fi
[ "$(cat "$temporary/log")" = 'Unsupported Linux architecture: armv7l' ]
[ ! -e microsctl ]
echo 'Installer platform tests passed.'
