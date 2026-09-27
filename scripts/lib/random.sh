#!/usr/bin/env bash

# Use finite random input so pipefail does not turn head's SIGPIPE into a failure.
# rand_hex takes a byte count; rand_password takes a character count.
rand_hex() {
  local bytes="${1:-48}"
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex "$bytes"
  else
    od -An -N "$bytes" -tx1 /dev/urandom | tr -d ' \n'
  fi
}

rand_password() {
  local length="${1:-24}" value="" chunk
  while [ "${#value}" -lt "$length" ]; do
    if command -v openssl >/dev/null 2>&1; then
      chunk="$(openssl rand -base64 "$length" | LC_ALL=C tr -dc 'A-Za-z0-9')" || return
    else
      chunk="$(dd if=/dev/urandom bs=256 count=1 2>/dev/null | LC_ALL=C tr -dc 'A-Za-z0-9')" || return
    fi
    value="${value}${chunk}"
  done
  printf '%s\n' "${value:0:$length}"
}
