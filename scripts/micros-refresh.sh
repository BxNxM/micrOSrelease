#!/bin/sh

set -eu

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$script_directory/.." && pwd)
cache_file="$repository_root/.micros-source"

source_root=${MICROS_SOURCE:-}
if [ -z "$source_root" ] && [ -f "$cache_file" ]; then
	IFS= read -r source_root < "$cache_file" || source_root=
fi

if [ -z "$source_root" ]; then
	printf 'micrOS source path: '
	IFS= read -r source_root
fi

if [ -z "$source_root" ]; then
	printf '%s\n' 'micrOS source path cannot be empty.' >&2
	exit 1
fi

case "$source_root" in
	/*) ;;
	*) source_root="$repository_root/$source_root" ;;
esac

if [ ! -d "$source_root" ]; then
	printf 'micrOS source directory does not exist: %s\n' "$source_root" >&2
	exit 1
fi
source_root=$(CDPATH= cd -- "$source_root" && pwd)

source_precompiled="$source_root/toolkit/workspace/precompiled"
source_modules="$source_precompiled/modules"
source_web="$source_precompiled/web"
source_firmware="$source_root/micrOS/micropython"
destination_modules="$repository_root/storage/modules/modules"
destination_web="$repository_root/storage/modules/web"
destination_frameworks="$repository_root/storage/frameworks"

for directory in "$source_modules" "$source_web" "$source_firmware"; do
	if [ ! -d "$directory" ]; then
		printf 'Required micrOS source directory does not exist: %s\n' "$directory" >&2
		exit 1
	fi
done

mkdir -p "$destination_modules" "$destination_web" "$destination_frameworks"

module_count=0
for destination in "$destination_modules"/*; do
	[ -f "$destination" ] || continue
	name=${destination##*/}
	if [ ! -f "$source_modules/$name" ]; then
		printf 'Precompiled module is missing: %s\n' "$source_modules/$name" >&2
		exit 1
	fi
	module_count=$((module_count + 1))
done

firmware_count=0
for source_file in "$source_firmware"/micrOS-*.bin; do
	[ -f "$source_file" ] || continue
	name=${source_file##*/}
	matched=false
	for framework_directory in "$destination_frameworks"/*; do
		[ -d "$framework_directory" ] || continue
		board=${framework_directory##*/}
		case "$name" in
			micrOS-"$board"-*.bin)
				matched=true
				break
				;;
		esac
	done
	if [ "$matched" = false ]; then
		printf 'No framework directory matches firmware image: %s\n' "$source_file" >&2
		exit 1
	fi
	firmware_count=$((firmware_count + 1))
done

if [ "$firmware_count" -eq 0 ]; then
	printf 'No micrOS firmware images found in: %s\n' "$source_firmware" >&2
	exit 1
fi

temporary_cache="$cache_file.tmp"
printf '%s\n' "$source_root" > "$temporary_cache"
mv "$temporary_cache" "$cache_file"

for framework_directory in "$destination_frameworks"/*; do
	[ -d "$framework_directory" ] || continue
	board=${framework_directory##*/}
	has_firmware=false
	for source_file in "$source_firmware"/micrOS-"$board"-*.bin; do
		if [ -f "$source_file" ]; then
			has_firmware=true
			break
		fi
	done
	[ "$has_firmware" = true ] || continue

	for source_file in "$source_firmware"/micrOS-"$board"-*.bin; do
		[ -f "$source_file" ] || continue
		cp -p "$source_file" "$framework_directory/"
	done
done

web_manifest=$(mktemp "${TMPDIR:-/tmp}/micros-web.XXXXXX")
trap 'rm -f "$web_manifest"' EXIT HUP INT TERM
(CDPATH= cd -- "$source_web" && find . -type f ! -path './.*' ! -path '*/.*' -print) > "$web_manifest"

web_count=0
while IFS= read -r source_entry; do
	relative_path=${source_entry#./}
	source_file="$source_web/$relative_path"
	destination_directory=$(dirname -- "$destination_web/$relative_path")
	mkdir -p "$destination_directory"
	cp -p "$source_file" "$destination_web/$relative_path"
	web_count=$((web_count + 1))
done < "$web_manifest"
rm -f "$web_manifest"
trap - EXIT HUP INT TERM

for destination in "$destination_modules"/*; do
	[ -f "$destination" ] || continue
	name=${destination##*/}
	cp -p "$source_modules/$name" "$destination"
done

printf 'Refreshed %s firmware images, %s selected modules, and %s web files from %s\n' \
	"$firmware_count" "$module_count" "$web_count" "$source_root"
