#!/usr/bin/env bash

namespace="default"
debug=""
appInstance="chaos-default-app"
appGroup="chaos-default-app-group"
port="19527"
endpoint=""
chaosagentRelease=""
uniqueAgentId=""
clusterId=""
clusterName=""

DEST_PATH="/opt/chaos"

DEST_FILENAME="chaosagent"
PID_FILE="/var/run/chaosagent.pid"
LOG_FILE_NAME="chaosagent.log"
FLAG_FILENAME=".flag"

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

checkEndpoint() {
  if [ -z ${endpoint} ]; then
    logWarn "Endpoint is empty."
    sendResult 1 "must specified endpoint"
  fi
}

checkChaosagentRelease() {
  if [ -z ${chaosagentRelease} ]; then
    logWarn "ChaosagentRelease is empty."
    sendResult 1 "must specified chaosagentRelease"
  fi
}

checkChaosagentBin() {
  local bin_file=${DEST_PATH}/${DEST_FILENAME}
  if [ ! -e ${bin_file} ]; then
    logWarn "chaosagent bin not found"
    sendResult 1 "chaosagent bin not found"
  fi
}

checkDestPath() {
  if [ ! -d ${DEST_PATH} ]; then
    mkdir ${DEST_PATH}
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
An chaosagent controller

Commands:
  install   Install chaosagent. If the chaosagent already exists, it will be stopped, deleted and reinstalled
  uninstall Uninstall chaosagent. If the chaosagent is running, it will be stopped and deleted.
  start     Start chaosagent，skip if it is already started. If you want restart it, please use restart command
  stop      Stop chaosagent
  restart   Restart chaosagent

Install Command Options:
  -n namespace            Namespace where the current ecs is located, default is default
  -d debug                Enable debug
  -p applicationInstance  The application instance name to which the specified machine belongs，default is chaos-default-app.
  -g applicationGroup     The application group to which the specified machine belongs,default is chaos-default-app-group.
  -P port                 chaosagent Agent listen port, default is 19527
  -t endpoint             chaos-platform endpoint
  -r chaosagentRelease    chaosagent release url
  -u uniqueAgentId        Is also probes id
  -i clusterId            Kubernetes cluster id
  -N clusterName          kubernetes cluster name
"
  sendResult 3 "illegal parameters"
}

createFlags() {
  local start_f="$1"
  local flags=""
  if [ -n "${namespace}" ]; then
    flags="${flags} --namespace ${namespace}"
  fi
  if [ -n "${scope}" ]; then
    flags="${flags} --scope ${scope}"
  fi
  if [ -n "${debug}" ]; then
    flags="${flags} --prof --debug=true"
  fi
  if [ -n "${appInstance}" ]; then
    flags="${flags} --appInstance ${appInstance}"
  fi
  if [ -n "${appGroup}" ]; then
    flags="${flags} --appGroup ${appGroup}"
  fi
  if [ -n "${port}" ]; then
    flags="${flags} --port ${port}"
  fi
  if [ -n "${uniqueAgentId}" ]; then
    flags="${flags} --agentId ${uniqueAgentId}"
  fi
  if [ -n "${clusterId}" ]; then
    flags="${flags} --kubernetes.cluster.id ${clusterId}"
  fi
  if [ -n "${clusterName}" ]; then
    flags="${flags} --kubernetes.cluster.name ${clusterName}"
  fi
  flags="${flags} --transport.endpoint ${endpoint}"
  # record flag
  echo "${flags}" >${DEST_PATH}/${FLAG_FILENAME} | tr '\n' ' '
  echo "${flags}" | tr '\n' ' '
}

start0() {
  local start_flag="$1"
  local pid=$(pgrep -f chaos/chaosagent)
  if [ -n "${pid}" ]; then
    echo "0,chaosagent has been started" | tr '\n' ' '
    return
  fi
  flags=$(createFlags ${start_flag})
  nohup ${DEST_PATH}/${DEST_FILENAME} ${flags} >>${DEST_PATH}/${LOG_FILE_NAME} 2>&1 &
  local count=10
  while :; do
    if [ "${count}" = "0" ]; then
      if [ -e "${DEST_PATH}/${LOG_FILE_NAME}" ];then
      logWarn `cat ${DEST_PATH}/${LOG_FILE_NAME}`
      fi
      stop >/dev/null 2>&1
      echo "1,start timeout" | tr '\n' ' '
      return
    fi

    if [ -e "${PID_FILE}" ]; then
      if [ $(cat "${PID_FILE}") = "-1" ]; then
        logWarn "[start] failed:"`cat ${DEST_PATH}/${LOG_FILE_NAME}`
        echo "1,start failed" | tr '\n' ' '
        return
      else
        break
      fi
    else
      count=$((count - 1))
      sleep 1
    fi
  done
  echo "0,success" | tr '\n' ' '
}

start() {
  logInfo "[+starting] chaosagent is starting..."
  local result=$(start0 ${StartNewFlag})
  code=${result%,*}
  message=${result##*,}
  if [ "${code}" = "0" ]; then
    sendResult 0 "${message}"
  else
    sendResult "${code}" "${message}"
  fi
}

stop() {
  local process="chaos/chaosagent"
  pid=$(ps -ef | grep ${process} | grep -v grep | awk '{print $2}')
  if [ -n "${pid}" ]; then
    kill -15 ${pid} >/dev/null 2>&1
    sleep 2
  fi
  pkill -9 -f "${process}"
  rm_rf ${PID_FILE}
  logInfo "[+stop] chaosagent is stopped."
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

install() {
  download ${chaosagentRelease} ${DEST_PATH}/${DEST_FILENAME}
  chmod +x ${DEST_PATH}/${DEST_FILENAME}
}

uninstall() {
  rm_rf ${DEST_PATH}
}

uninstall_check() {
  local pid=$(pgrep -f chaos/chaosagent)
  if [ -n "${pid}" ]; then
    sendResult 1 "uninstall failed"
  else
    sendResult 0 "uninstall successfully"
  fi
}

action="$1"
shift

while getopts ":n:s:m:d:p:g:P:t:o:r:u:i:N" opt; do
  case ${opt} in
  n)
    namespace="${OPTARG}"
    ;;
  s)
    scope="${OPTARG}"
    ;;
  d)
    debug="true"
    ;;
  m)
    startupMode="${OPTARG}"
    ;;
  p)
    appInstance="${OPTARG}"
    ;;
  g)
    appGroup="${OPTARG}"
    ;;
  P)
    port="${OPTARG}"
    ;;
  t)
    endpoint="${OPTARG}"
    ;;
  o)
    protocol="${OPTARG}"
    ;;
  r)
    chaosagentRelease="${OPTARG}"
    ;;
  u)
    uniqueAgentId="${OPTARG}"
    ;;
  i)
    clusterId="${OPTARG}"
    ;;
  N)
    clusterName="${OPTARG}"
    ;;
  \?) ;;

  esac
done

case "${action}" in
install)
  stop
  checkRoot
  checkChaosagentRelease
  checkDestPath
  checkEndpoint
  install
  checkChaosagentBin
  start
  ;;
uninstall)
  stop
  uninstall
  uninstall_check
  ;;
start)
  if [ ! "${startupMode}" = ${crontabMode} ]; then
    deployChaosTools
    checkRoot
    checkEndpoint
  fi
  checkChaosagentBin
  start
  ;;
stop)
  stop
  ;;
restart)
  stop
  checkChaosagentBin
  start
  ;;
*)
  usages
  ;;
esac
logInfo "[${action}] execute successfully"
exit 0
