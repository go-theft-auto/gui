#!/usr/bin/env bash
# Setup CGO environment for OpenGL/X11 compilation and runtime.
# Sources all -dev packages from nix store.
#
# Usage:
#   source scripts/setup-cgo.sh    # Sets env vars + generates .env.cgo
#   devbox run -- scripts/setup-cgo.sh  # Just generates .env.cgo for IDE

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${PROJECT_ROOT:-$(dirname "$SCRIPT_DIR")}"

INCLUDE_DIRS="${DEVBOX_PACKAGES_DIR:-/tmp}/include"
LIB_DIRS="${DEVBOX_PACKAGES_DIR:-/tmp}/lib"
PKG_DIRS="${DEVBOX_PACKAGES_DIR:-/tmp}/lib/pkgconfig"

for pkg in libx11 libglvnd libxrandr libxcursor libxi libxinerama libxrender libxfixes libxext; do
    dev_path=$(ls -d /nix/store/*${pkg}*-dev 2>/dev/null | head -1)
    if [ -n "$dev_path" ]; then
        INCLUDE_DIRS="$INCLUDE_DIRS:$dev_path/include"
        LIB_DIRS="$LIB_DIRS:$dev_path/lib"
        [ -d "$dev_path/lib/pkgconfig" ] && PKG_DIRS="$PKG_DIRS:$dev_path/lib/pkgconfig"
    fi
done

RUNTIME_LIB_DIRS="${DEVBOX_PACKAGES_DIR:-/tmp}/lib"
for pkg in libx11 libglvnd libxrandr libxcursor libxi libxinerama libxrender libxfixes libxext; do
    runtime_path=$(ls -d /nix/store/*-${pkg}-*/lib 2>/dev/null | grep -v "\-dev" | head -1)
    if [ -n "$runtime_path" ]; then
        RUNTIME_LIB_DIRS="$RUNTIME_LIB_DIRS:$runtime_path"
    fi
done

RPATH_FLAGS=""
IFS=':' read -ra RUNTIME_PATHS <<< "$RUNTIME_LIB_DIRS"
for rpath in "${RUNTIME_PATHS[@]}"; do
    if [ -n "$rpath" ] && [ -d "$rpath" ]; then
        RPATH_FLAGS="$RPATH_FLAGS -Wl,-rpath,$rpath"
    fi
done

export CGO_CFLAGS="-I${INCLUDE_DIRS//:/ -I}"
export CGO_LDFLAGS="-L${LIB_DIRS//:/ -L}$RPATH_FLAGS"
export PKG_CONFIG_PATH="$PKG_DIRS"

ENV_FILE="${PROJECT_ROOT:-.}/.env.cgo"
cat > "$ENV_FILE" << EOF
CGO_CFLAGS="${CGO_CFLAGS}"
CGO_LDFLAGS="${CGO_LDFLAGS}"
PKG_CONFIG_PATH="${PKG_DIRS}"
PKG_CONFIG_PATH_x86_64_unknown_linux_gnu="${PKG_DIRS}"
LD_LIBRARY_PATH="${RUNTIME_LIB_DIRS}${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
DISPLAY="${DISPLAY:-:0}"
XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
EOF
echo "Generated $ENV_FILE"

export PKG_CONFIG_PATH_x86_64_unknown_linux_gnu="$PKG_DIRS"
export LD_LIBRARY_PATH="${RUNTIME_LIB_DIRS}${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
