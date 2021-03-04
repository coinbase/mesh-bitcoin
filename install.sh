#!/bin/sh
# Copyright 2020 Coinbase, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# Inspired by: https://github.com/golangci/golangci-lint/blob/master/install.sh

usage() {
  this=$1
  cat <<EOF
$this: download pre-compiled Docker images for coinbase/rosetta-defichain 

Usage: $this [-d]
  -d turns on debug logging

EOF
  exit 2
}

parse_args() {
  while getopts "dh?" arg; do
    case "$arg" in
      d) log_set_priority 10 ;;
      h | \?) usage "$0" ;;
    esac
  done
  shift $((OPTIND - 1))
  TAG=$1
}
execute() {
  tmpdir=$(mktemp -d)
  log_info "downloading image into ${tmpdir}"
  http_download "${tmpdir}/${TARBALL}" "${TARBALL_URL}" "" "1"
  docker load --input "${tmpdir}/${TARBALL}"
  docker tag "rosetta-defichain:${TAG}" "rosetta-defichain:latest"
  log_info "loaded rosetta-defichain:${TAG} and tagged as rosetta-defichain:latest"
  rm -rf "${tmpdir}"
  log_info "removed temporary directory ${tmpdir}"
}
github_tag() {
  log_info "checking GitHub for latest tag"
  REALTAG=$(github_release "$OWNER/$REPO" "${TAG}")
  TAG="$REALTAG"
}

cat /dev/null <<EOF
------------------------------------------------------------------------
https://github.com/client9/shlib - portable posix shell functions
Public domain - http://unlicense.org
https://github.com/client9/shlib/blob/master/LICENSE.md
but credit (and pull requests) appreciated.
------------------------------------------------------------------------
EOF
is_command() {
  command -v "$1" >/dev/null
}
echoerr() {
  echo "$@" 1>&2
}
log_prefix() {
  echo "$0"
}
_logp=6
log_set_priority() {
  _logp="$1"
}
log_priority() {
  if test -z "$1"; then
    echo "$_logp"
    return
  fi
  [ "$1" -le "$_logp" ]
}
log_tag() {
  case $1 in
    0) echo "emerg" ;;
    1) echo "alert" ;;
    2) echo "crit" ;;
    3) echo "err" ;;
    4) echo "warning" ;;
    5) echo "notice" ;;
    6) echo "info" ;;
    7) echo "debug" ;;
    *) echo "$1" ;;
  esac
}
log_debug() {
  log_priority 7 || return 0
  echoerr "$(log_prefix)" "$(log_tag 7)" "$@"
}
log_info() {
  log_priority 6 || return 0
  echoerr "$(log_prefix)" "$(log_tag 6)" "$@"
}
log_err() {
  log_priority 3 || return 0
  echoerr "$(log_prefix)" "$(log_tag 3)" "$@"
}
log_crit() {
  log_priority 2 || return 0
  echoerr "$(log_prefix)" "$(log_tag 2)" "$@"
}
untar() {
  tarball=$1
  case "${tarball}" in
    *.tar.gz | *.tgz) tar --no-same-owner -xzf "${tarball}" ;;
    *.tar) tar --no-same-owner -xf "${tarball}" ;;
    *.zip) unzip "${tarball}" ;;
    *)
      log_err "untar unknown archive format for ${tarball}"
      return 1
      ;;
  esac
}
http_download_curl() {
  local_file=$1
  source_url=$2
  header=$3
  loud=$4
  quiet_var="-L"
  if [ -z "$loud" ]; then
    quiet_var="-sL"
  fi
  if [ -z "$header" ]; then
    code=$(curl -w '%{http_code}' "$quiet_var" -o "$local_file" "$source_url")
  else
    code=$(curl -w '%{http_code}' "$quiet_var" -H "$header" -o "$local_file" "$source_url")
  fi
  if [ "$code" != "200" ]; then
    log_debug "http_download_curl received HTTP status $code"
    return 1
  fi
  return 0
}
http_download_wget() {
  local_file=$1
  source_url=$2
  header=$3
  loud=$4
  quiet_var=""
  if [ -z "$loud" ]; then
    quiet_var="-q"
  fi

  if [ -z "$header" ]; then
    wget "$quiet_var" -O "$local_file" "$source_url"
  else
    wget "$quiet_var" --header "$header" -O "$local_file" "$source_url"
  fi
}
http_download() {
  log_debug "http_download $2"
  if is_command curl; then
    http_download_curl "$@"
    return
  elif is_command wget; then
    http_download_wget "$@"
    return
  fi
  log_crit "http_download unable to find wget or curl"
  return 1
}
http_copy() {
  tmp=$(mktemp)
  http_download "${tmp}" "$1" "$2" || return 1
  body=$(cat "$tmp")
  rm -f "${tmp}"
  echo "$body"
}
github_release() {
  owner_repo=$1
  version=$2
  test -z "$version" && version="latest"
  giturl="https://github.com/${owner_repo}/releases/${version}"
  json=$(http_copy "$giturl" "Accept:application/json")
  test -z "$json" && return 1
  version=$(echo "$json" | tr -s '\n' ' ' | sed 's/.*"tag_name":"//' | sed 's/".*//')
  test -z "$version" && return 1
  echo "$version"
}
cat /dev/null <<EOF
------------------------------------------------------------------------
End of functions from https://github.com/client9/shlib
------------------------------------------------------------------------
EOF

BINARY=rosetta-defichain
FORMAT=tar.gz
OWNER=coinbase
REPO="rosetta-defichain"
PREFIX="$OWNER/$REPO"

# use in logging routines
log_prefix() {
	echo "$PREFIX"
}
GITHUB_DOWNLOAD=https://github.com/${OWNER}/${REPO}/releases/download

parse_args "$@"

github_tag

log_info "found version: ${TAG}"

NAME=${BINARY}-${TAG}
TARBALL=${NAME}.${FORMAT}
TARBALL_URL=${GITHUB_DOWNLOAD}/${TAG}/${TARBALL}

execute
