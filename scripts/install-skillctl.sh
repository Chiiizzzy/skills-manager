#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/install-skillctl.sh [options]

Build skillctl and register it as a shell command.

Options:
  --bin-dir DIR   Directory to place the skillctl command link.
                  Defaults to $SKILLCTL_BIN_DIR or ~/.local/bin.
  --name NAME     Command name to register. Defaults to skillctl.
  --no-build      Skip go build and link the existing bin/skillctl.
  --force         Replace an existing command at the target path.
  -h, --help      Show this help.

Examples:
  scripts/install-skillctl.sh
  SKILLCTL_BIN_DIR=/usr/local/bin scripts/install-skillctl.sh
  scripts/install-skillctl.sh --bin-dir /usr/local/bin --force
EOF
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

bin_dir="${SKILLCTL_BIN_DIR:-"$HOME/.local/bin"}"
command_name="${SKILLCTL_NAME:-skillctl}"
build=1
force=0

while (($#)); do
  case "$1" in
    --bin-dir)
      if [[ $# -lt 2 || -z "$2" ]]; then
        echo "error: --bin-dir requires a directory" >&2
        exit 2
      fi
      bin_dir="$2"
      shift 2
      ;;
    --name)
      if [[ $# -lt 2 || -z "$2" ]]; then
        echo "error: --name requires a command name" >&2
        exit 2
      fi
      command_name="$2"
      shift 2
      ;;
    --no-build)
      build=0
      shift
      ;;
    --force)
      force=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ "$command_name" == */* || -z "$command_name" ]]; then
  echo "error: command name must not be empty or contain '/'" >&2
  exit 2
fi

target="$repo_root/bin/skillctl"
link="$bin_dir/$command_name"

if ((build)); then
  if ! command -v go >/dev/null 2>&1; then
    echo "error: go is required to build skillctl" >&2
    exit 1
  fi

  mkdir -p "$repo_root/bin"
  (cd "$repo_root" && go build -o "$target" ./cmd/skillctl)
elif [[ ! -x "$target" ]]; then
  echo "error: $target does not exist or is not executable; rerun without --no-build" >&2
  exit 1
fi

mkdir -p "$bin_dir"

if [[ -e "$link" || -L "$link" ]]; then
  existing=""
  if [[ -L "$link" ]]; then
    existing="$(readlink "$link")"
  fi

  if [[ "$existing" != "$target" ]]; then
    if ((force)); then
      rm -f "$link"
    else
      echo "error: $link already exists; use --force to replace it" >&2
      exit 1
    fi
  fi
fi

ln -sfn "$target" "$link"

echo "Installed $command_name -> $target"

case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *)
    cat <<EOF

Warning: $bin_dir is not in PATH.
Add this to your shell profile, then restart the shell:

  export PATH="$bin_dir:\$PATH"
EOF
    ;;
esac
