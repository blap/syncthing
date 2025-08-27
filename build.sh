#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

script() {
	name="$1"
	shift
	go run "script/$name.go" "$@"
}

build() {
	# Disable CGO for Windows builds to avoid the modernc.org/libc issue
	# See docs/windows-cgo-build-guide.md for more details about Windows builds
	if [[ "${GOOS:-}" == "windows" ]] || [[ "$(uname -s 2>/dev/null || echo '')" == MINGW* ]] || [[ "$(uname -s 2>/dev/null || echo '')" == MSYS* ]]; then
		# Check if force-cgo flag is set
		FORCE_CGO=false
		for arg in "$@"; do
			if [[ "$arg" == "-force-cgo" ]] || [[ "$arg" == "--force-cgo" ]]; then
				FORCE_CGO=true
				break
			fi
		done
		
		if [[ "$FORCE_CGO" == false ]]; then
			export CGO_ENABLED=0
		else
			export CGO_ENABLED=1
		fi
		
		# Ensure goversioninfo is available for Windows builds
		if ! command -v goversioninfo &> /dev/null; then
			echo "Installing goversioninfo tool..."
			go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
		fi
		
		# Ensure GOPATH/bin is in PATH
		GOPATH_BIN="$(go env GOPATH)/bin"
		if [[ ":$PATH:" != *":$GOPATH_BIN:"* ]]; then
			export PATH="$GOPATH_BIN:$PATH"
		fi
	fi
	go run build.go "$@"
}

case "${1:-default}" in
	test)
		LOGGER_DISCARD=1 build test
		;;

	bench)
		LOGGER_DISCARD=1 build bench
		;;

	prerelease)
		script authors
		script copyrights
		build weblate
		pushd man ; ./refresh.sh ; popd
		git add -A gui man AUTHORS
		git commit -m 'chore(gui, man, authors): update docs, translations, and contributors'
		;;

	*)
		build "$@"
		;;
esac