#!/usr/bin/env bash

toolsName="chaosblade"
version="latest"
toolsRelease=""

DEST_PATH="/opt"

green() {
  M=$1
  shift
  echo -e "\033[32m$M \033[0m $*"
}

red() {
  M=$1
  shift
  echo -e "\033[31m$M \033[0m $*"
}

logTTY() {
  LEVEL=$1
  MSG=$2
  shift

  if [ "$LEVEL" = "WARN" ]; then
    red "$(date +'%Y-%m-%d %T') [$LEVEL]" "$@"
  elif [ "$LEVEL" = "INFO" ]; then
    green "$(date +'%Y-%m-%d %T') [$LEVEL]" "$@"
  else
    echo -e "$(date +'%Y-%m-%d %T') [$LEVEL]" "$@"
  fi
}

log() {
  if [ -t 0 ]; then
    logTTY "$@"
  fi
}

logWarn() {
  log 'WARN' "$@"
}

logInfo() {
  log 'INFO' "$@"
}

sendResult() {
  local exitCode="$1"
  local msg="$2"
  if [ "${exitCode}" = 0 ]; then
    logInfo "${msg}"
  else
    logWarn "${msg}"
  fi
  exit "${exitCode}"
}

checkRoot() {
  if [ ! -w "/root" ]; then
    logWarn "Must have root folder permission, exit."
    sendResult 1 "no root folder permission"
  fi
}

checkUser() {
  name=$(whoami)
  if [ "$name" != "root" ]; then
    logWarn "The script must be executed by root, exit."
    sendResult 1 "The script must be executed by root"
  fi
}

checkToolsName() {
  if [ -z ${toolsName} ]; then
    logWarn "tools name is empty."
    sendResult 1 "must specified tools name "
  fi
}

checkVersion() {
  if [ -z ${version} ]; then
    logWarn "version is empty."
    sendResult 1 "must specified version"
  fi
}

checkInstallPath() {
  if [ ! -f "${DEST_PATH}/${toolsName}" ]; then
    mk_dir "${DEST_PATH}/${toolsName}"
  fi
}

checkToolsRelease() {
  if [ -z ${toolsRelease} ]; then
    logWarn "toolsRelease is empty."
    sendResult 1 "must specified toolsRelease"
  fi
}

checkDestPath() {
  if [ ! -d ${DEST_PATH} ]; then
    mk_dir ${DEST_PATH}
  fi
}

rm_rf() {
  if [ ! -e "$1" ]; then
    return
  fi
  if [ ! -w "$1" ]; then
    logWarn "[-rm err] $1, no permission"
    return
  fi
  if [ -f "$1" ]; then
    rm -rf "$1"
    logInfo "[-rm f] $1"
  elif [ -d "$1" ]; then
    rm -rf "$1"
    logInfo "[-rm d] $1"
  fi
}

mk_dir() {
  if [ ! -d "$1" ]; then
    mkdir -p "$1"
    logInfo "[+mkdir] $1"
  fi
}

usages() {
  echo "
Usage: sh $0 COMMAND [OPTIONS]
An chaos tools controller

Commands:
  install   Install chaos tools. If the chaos tools already exists, it will be deleted and reinstalled
  uninstall Uninstall chaos tools. If the chaos tools not exists, noting to do.

Install Command Options:
  -n toolsName      chaos tools name
  -v version        chaos tools version
  -r toolsRelease   chaos tools release url
"
  sendResult 3 "illegal parameters"
}

download() {
  local originFile="$1"
  logInfo "[+downloading] ${originFile##*/} to [$2]"
  wget -q "$1" -O "$2"
  local ecode=$?
  if [ "${ecode}" != 0 ]; then
    sendResult ${ecode} "download ${originFile##*/} failed."
  fi
  logInfo "[+saved] ${originFile##*/} to [$2]"
}

unpackTar() {
  local originFile="$1"
  logInfo "[+unpacking] ${originFile##*/} to [$2]"
  tar -zxf "$1" -C "$2" --strip-components 1
  local ecode=$?
  if [ "${ecode}" != 0 ]; then
    sendResult ${ecode} "unpacking ${originFile##*/} failed."
  fi
  logInfo "[+saved] ${originFile##*/} to [$2]"
}

install() {
  download ${toolsRelease} "${DEST_PATH}/${toolsName}/$(echo ${toolsRelease} |awk -F '/' '{print $NF}')"
  unpackTar "${DEST_PATH}/${toolsName}/$(echo ${toolsRelease} |awk -F '/' '{print $NF}')" "${DEST_PATH}/${toolsName}"
}

uninstall() {
  rm_rf "${DEST_PATH}/${toolsName}"
}

action="$1"
shift

while getopts ":n:v:r:" opt; do
  case ${opt} in
  n)
    toolsName="${OPTARG}"
    ;;
  v)
    version="${OPTARG}"
    ;;
  r)
    toolsRelease="${OPTARG}"
    ;;
  \?) ;;
  esac
done

case "${action}" in
install)
  checkRoot
  checkUser
  checkToolsName
  checkVersion
  checkToolsRelease
  checkInstallPath
  install
  ;;
uninstall)
  uninstall
  ;;
*)
  usages
  ;;
esac
logInfo "[${action}] execute successfully"
exit 0
