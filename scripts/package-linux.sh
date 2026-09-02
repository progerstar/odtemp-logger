#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
source "${script_dir}/common.sh"

if [[ "$(uname -s)" != Linux ]]; then
    printf '%s\n' 'Linux AppImages must be built on Linux.' >&2
    exit 2
fi

case "$(uname -m)" in
    x86_64|amd64)
        go_arch=amd64
        appimage_arch=x86_64
        ;;
    aarch64|arm64)
        go_arch=arm64
        appimage_arch=aarch64
        ;;
    *)
        printf 'Unsupported Linux architecture: %s\n' "$(uname -m)" >&2
        exit 2
        ;;
esac

require_command awk file go install ldd
version=$(app_version)
build_root=${BUILD_ROOT:-"${release_root_dir}/build/linux-${appimage_arch}"}
dist_root=${1:-"${release_root_dir}/dist/linux"}

appimagetool_candidate=${APPIMAGETOOL:-appimagetool}
if [[ "${appimagetool_candidate}" == */* ]]; then
    [[ -x "${appimagetool_candidate}" ]] || {
        printf 'appimagetool is not executable: %s\n' "${appimagetool_candidate}" >&2
        exit 2
    }
    appimagetool_bin=${appimagetool_candidate}
else
    appimagetool_bin=$(command -v "${appimagetool_candidate}" || true)
    [[ -n "${appimagetool_bin}" ]] || {
        printf '%s\n' 'AppImage packaging requires appimagetool.' >&2
        printf '%s\n' 'Install it or set APPIMAGETOOL=/absolute/path/to/appimagetool.' >&2
        exit 2
    }
fi

appimagetool_args=(--no-appstream)
if [[ -n "${APPIMAGE_RUNTIME_FILE:-}" ]]; then
    [[ -f "${APPIMAGE_RUNTIME_FILE}" ]] || {
        printf 'AppImage runtime does not exist: %s\n' "${APPIMAGE_RUNTIME_FILE}" >&2
        exit 2
    }
    appimagetool_args+=(--runtime-file "${APPIMAGE_RUNTIME_FILE}")
fi

[[ -f "${ODTEMP_ICON_SVG}" ]] || {
    printf 'Application icon was not found: %s\n' "${ODTEMP_ICON_SVG}" >&2
    exit 1
}

mkdir -p "${build_root}" "${dist_root}"
binary="${build_root}/${ODTEMP_NAME}"
printf '==> Building %s %s for linux/%s\n' \
    "${ODTEMP_NAME}" "${version}" "${go_arch}"
(
    cd "${release_root_dir}"
    CGO_ENABLED=1 GOOS=linux GOARCH="${go_arch}" \
        go build -trimpath -buildvcs=false \
        -ldflags '-s -w' \
        -o "${binary}" .
)

[[ -x "${binary}" ]] || {
    printf 'Linux binary was not created: %s\n' "${binary}" >&2
    exit 1
}
verify_help_version "${binary}" "${version}"

if ldd "${binary}" | grep -q 'not found'; then
    printf '%s\n' 'The Linux binary has unresolved shared-library dependencies:' >&2
    ldd "${binary}" >&2
    exit 1
fi

package_dir=$(mktemp -d "${TMPDIR:-/tmp}/odtemp-logger-appdir.XXXXXX")
image_tmp=
cleanup()
{
    if [[ -n "${package_dir:-}" && "${package_dir}" == *odtemp-logger-appdir.* ]]; then
        rm -rf -- "${package_dir}"
    fi
    if [[ -n "${image_tmp:-}" && -f "${image_tmp}" ]]; then
        rm -f -- "${image_tmp}"
    fi
}
trap cleanup EXIT

mkdir -p \
    "${package_dir}/usr/bin" \
    "${package_dir}/usr/lib" \
    "${package_dir}/usr/share/applications" \
    "${package_dir}/usr/share/doc/${ODTEMP_NAME}" \
    "${package_dir}/usr/share/icons/hicolor/scalable/apps"
install -m 0755 "${binary}" "${package_dir}/usr/bin/${ODTEMP_NAME}"
install -m 0644 "${release_root_dir}/README.md" \
    "${package_dir}/usr/share/doc/${ODTEMP_NAME}/README.md"
install -m 0644 "${ODTEMP_ICON_SVG}" \
    "${package_dir}/${ODTEMP_NAME}.svg"
install -m 0644 "${ODTEMP_ICON_SVG}" \
    "${package_dir}/usr/share/icons/hicolor/scalable/apps/${ODTEMP_NAME}.svg"
ln -s "${ODTEMP_NAME}.svg" "${package_dir}/.DirIcon"

is_base_system_library()
{
    case "$1" in
        ld-linux*.so*|libanl.so*|libc.so*|libdl.so*|libm.so*|libnss_*.so*|libpthread.so*|libresolv.so*|librt.so*|libutil.so*)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

copy_elf_dependencies()
{
    local pass elf dependency name copied

    for pass in 1 2 3 4 5 6 7 8; do
        copied=0
        while IFS= read -r -d '' elf; do
            file "${elf}" | grep -q ELF || continue
            while IFS= read -r dependency; do
                [[ -f "${dependency}" ]] || continue
                name=$(basename -- "${dependency}")
                is_base_system_library "${name}" && continue
                if [[ ! -e "${package_dir}/usr/lib/${name}" ]]; then
                    cp -L "${dependency}" "${package_dir}/usr/lib/${name}"
                    copied=1
                fi
            done < <(ldd "${elf}" 2>/dev/null | \
                awk '/=> \// { print $3 } /^[[:space:]]*\// { print $1 }')
        done < <(find "${package_dir}/usr/bin" "${package_dir}/usr/lib" \
            -type f -print0)
        [[ "${copied}" == 0 ]] && break
    done
}

copy_elf_dependencies

{
    printf '%s\n' '#!/bin/sh'
    printf '%s\n' 'set -e'
    printf '%s\n' 'app_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)'
    printf '%s\n' 'export PATH="${app_dir}/usr/bin:${PATH}"'
    printf '%s\n' 'export LD_LIBRARY_PATH="${app_dir}/usr/lib${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}"'
    printf 'exec "${app_dir}/usr/bin/%s" "$@"\n' "${ODTEMP_NAME}"
} > "${package_dir}/AppRun"
chmod 0755 "${package_dir}/AppRun"

desktop_file="${package_dir}/${ODTEMP_NAME}.desktop"
{
    printf '%s\n' '[Desktop Entry]'
    printf '%s\n' 'Type=Application'
    printf 'Name=%s\n' "${ODTEMP_DISPLAY_NAME}"
    printf 'Comment=%s\n' "${ODTEMP_DESCRIPTION}"
    printf 'Exec=%s\n' "${ODTEMP_NAME}"
    printf 'Icon=%s\n' "${ODTEMP_NAME}"
    printf '%s\n' 'Categories=Utility;'
    printf '%s\n' 'Terminal=false'
    printf '%s\n' 'StartupNotify=true'
} > "${desktop_file}"
chmod 0644 "${desktop_file}"
cp "${desktop_file}" "${package_dir}/usr/share/applications/"
if command -v desktop-file-validate >/dev/null 2>&1; then
    desktop-file-validate "${desktop_file}"
fi

image="${dist_root}/${ODTEMP_NAME}-${version}-linux-${appimage_arch}.AppImage"
image_tmp="${dist_root}/.${ODTEMP_NAME}-${version}-linux-${appimage_arch}.AppImage"
rm -f -- "${image_tmp}"
(
    export ARCH=${appimage_arch}
    export APPIMAGE_EXTRACT_AND_RUN=${APPIMAGE_EXTRACT_AND_RUN:-1}
    export VERSION=${version}
    "${appimagetool_bin}" "${appimagetool_args[@]}" \
        "${package_dir}" "${image_tmp}"
)
[[ -f "${image_tmp}" ]] || {
    printf 'appimagetool did not create %s\n' "${image_tmp}" >&2
    exit 1
}
chmod 0755 "${image_tmp}"
if ! help_output=$(APPIMAGE_EXTRACT_AND_RUN=1 "${image_tmp}" -h 2>&1); then
    printf '%s\n' 'Could not run the AppImage help smoke test.' >&2
    exit 1
fi
[[ "${help_output}" == *"Версия: ${version}"* ]] || {
    printf 'AppImage help does not contain version %s.\n' "${version}" >&2
    exit 1
}
mv -f -- "${image_tmp}" "${image}"
image_tmp=

file "${image}"
printf 'Created %s\n' "${image}"
write_sha256 "${image}"

rm -rf -- "${package_dir}"
package_dir=
trap - EXIT
