#!/bin/sh
set -e

REPO="lovincyrus/personal-vault"
BINARY="pvault"

main() {
    os=$(detect_os)
    arch=$(detect_arch)

    version=$(fetch_latest_version)
    echo "Installing ${BINARY} ${version} (${os}/${arch})..."

    tarball="${BINARY}_${version#v}_${os}_${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/${version}/${tarball}"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "${tmpdir}"' EXIT

    echo "Downloading ${url}..."
    curl -fsSL "${url}" -o "${tmpdir}/${tarball}"
    tar -xzf "${tmpdir}/${tarball}" -C "${tmpdir}"

    install_dir=$(choose_install_dir)
    echo "Installing to ${install_dir}..."
    mkdir -p "${install_dir}"
    cp "${tmpdir}/${BINARY}" "${install_dir}/${BINARY}"
    chmod +x "${install_dir}/${BINARY}"

    if "${install_dir}/${BINARY}" help > /dev/null 2>&1; then
        echo "Installed ${BINARY} ${version} to ${install_dir}/${BINARY}"
    else
        echo "Warning: ${BINARY} installed but could not verify. Check your PATH."
    fi
}

detect_os() {
    case "$(uname -s)" in
        Darwin) echo "darwin" ;;
        Linux)  echo "linux" ;;
        *)      echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)  echo "amd64" ;;
        amd64)   echo "amd64" ;;
        aarch64) echo "arm64" ;;
        arm64)   echo "arm64" ;;
        *)       echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
    esac
}

fetch_latest_version() {
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

    if [ -z "${version}" ]; then
        echo "Could not determine latest version" >&2
        exit 1
    fi
    echo "${version}"
}

choose_install_dir() {
    if [ -w /usr/local/bin ]; then
        echo "/usr/local/bin"
    else
        dir="${HOME}/.local/bin"
        case ":${PATH}:" in
            *":${dir}:"*) ;;
            *) echo "Note: add ${dir} to your PATH" >&2 ;;
        esac
        echo "${dir}"
    fi
}

main
