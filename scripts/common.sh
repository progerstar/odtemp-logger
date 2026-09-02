#!/usr/bin/env bash

if ((BASH_VERSINFO[0] < 3)); then
    printf '%s\n' 'odtemp-logger packaging scripts require Bash 3 or newer.' >&2
    if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
        return 2
    fi
    exit 2
fi

release_script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
release_root_dir=$(cd -- "${release_script_dir}/.." && pwd)

readonly ODTEMP_NAME='odtemp-logger'
readonly ODTEMP_DISPLAY_NAME='ODTemp Logger'
readonly ODTEMP_DESCRIPTION='Monitor temperature and humidity from ODTEMP-1 USB sensors'
readonly ODTEMP_BUNDLE_ID='com.opendev.odtemp-logger'
readonly ODTEMP_MACOS_MIN_VERSION='12.0'
readonly ODTEMP_ICON_PNG="${release_root_dir}/Icon.png"
readonly ODTEMP_ICON_SVG="${release_root_dir}/Icon.svg"

require_command()
{
    local command_name

    for command_name in "$@"; do
        if ! command -v "${command_name}" >/dev/null 2>&1; then
            printf 'Required command was not found: %s\n' "${command_name}" >&2
            return 2
        fi
    done
}

app_version()
{
    local version

    version=$(awk -F '"' \
        '/^[[:space:]]*VERSION[[:space:]]*=/ { print $2; exit }' \
        "${release_root_dir}/main.go")
    if [[ ! "${version}" =~ ^[0-9]+(\.[0-9]+){1,2}$ ]]; then
        printf 'Invalid application version in main.go: %s\n' \
            "${version:-<empty>}" >&2
        return 1
    fi

    printf '%s\n' "${version}"
}

verify_help_version()
{
    local executable=$1
    local version=$2
    local help_output

    if ! help_output=$("${executable}" -h 2>&1); then
        printf 'Could not run %s -h.\n' "${executable}" >&2
        return 1
    fi
    if [[ "${help_output}" != *"Версия: ${version}"* ]]; then
        printf 'Application help does not contain version %s.\n' "${version}" >&2
        return 1
    fi
}

write_sha256()
{
    local artifact=$1
    local artifact_dir artifact_name

    [[ -f "${artifact}" ]] || {
        printf 'Cannot checksum missing artifact: %s\n' "${artifact}" >&2
        return 1
    }

    artifact_dir=$(cd -- "$(dirname -- "${artifact}")" && pwd)
    artifact_name=$(basename -- "${artifact}")
    (
        cd "${artifact_dir}"
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum "${artifact_name}" > "${artifact_name}.sha256"
        elif command -v shasum >/dev/null 2>&1; then
            shasum -a 256 "${artifact_name}" > "${artifact_name}.sha256"
        else
            printf '%s\n' 'Neither sha256sum nor shasum is available.' >&2
            return 2
        fi
    )
    printf 'Created %s.sha256\n' "${artifact}"
}
