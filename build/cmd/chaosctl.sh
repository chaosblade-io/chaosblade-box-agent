#!/usr/bin/env bash

namespace="default"
license=""
appInstance="chaos-default-app"
appGroup="chaos-default-app-group"
port="19527"
endpoint=""

DEST_PATH=$(
  cd "$(dirname "$0")"
  pwd
)
DEST_FILENAME="agent"
PID_FILE="/var/run/chaos.pid"
LOG_FILE_NAME="agent.log"
FLAG_FILENAME=".flag"

# value: manual|console|crontab
manualMode="manual"
crontabMode="crontab"
startupMode=${manualMode}

RECOVERY_SCRIPT_NAME="chaosrcv.sh"
CTL_SCRIPT_NAME="chaosctl.sh"

StartOldFlag="old"
StartNewFlag="new"

crontabCmd="*/3 * * * * root ${DEST_PATH}/${RECOVERY_SCRIPT_NAME} > /dev/null 2>&1"
crontabPath="/etc/cron.d"
crontabFileName="chaosCrontab"

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

log_tty() {
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
    log_tty "$@"
  fi
}

log_warn() {
  log 'WARN' "$@"
}

log_info() {
  log 'INFO' "$@"
}

sendResult() {
  local exitCode="$1"
  local msg="$2"
  if [ "${exitCode}" = 0 ]; then
    log_info "${msg}"
  else
    log_warn "${msg}"
  fi
  #    clean
  exit "${exitCode}"
}

check_root() {
  if [ ! -w "/root" ]; then
    log_warn "Must have root folder permission, exit."
    sendResult 1 "no root folder permission"
  fi
}

#checkUser
checkUser() {
  name=$(whoami)
  if [ "$name" != "root" ]; then
    log_warn "The script must be executed by root, exit."
    sendResult 1 "The script must be executed by root"
  fi
}

check_license() {
  if [ -z ${license} ]; then
    log_warn "License is empty."
    sendResult 1 "must pass license"
  fi
}
check_endpoint() {
  if [ -z ${endpoint} ]; then
    log_warn "Endpoint is empty."
    sendResult 1 "must endpoint"
  fi
}

checkChaosBin() {
  local bin_file=${DEST_PATH}/${DEST_FILENAME}
  if [ ! -e ${bin_file} ]; then
    log_warn "Chaos client bin not found"
    sendResult 1 "Chaos client bin not found"
  fi
}
rm_rf() {
  if [ ! -e "$1" ]; then
    return
  fi
  if [ ! -w "$1" ]; then
    log_warn "[-rm err] $1, no permission"
    return
  fi
  if [ -f "$1" ]; then
    rm -rf "$1"
    log_info "[-rm f] $1"
  elif [ -d "$1" ]; then
    rm -rf "$1"
    log_info "[-rm d] $1"
  fi
}

mk_dir() {
  if [ ! -d "$1" ]; then
    mkdir -p "$1"
    log_info "[+mkdir] $1"
  fi
}

usages() {
  echo "
Usage: sh $0 COMMAND [OPTIONS]
An CHAOS Client controller

Commands:
  install   Install chaos client from official site. If the client already exists, it will be stopped, deleted and reinstalled
  uninstall Uninstall the client. If the client is running, it will be stopped and deleted.
  start     Start the client，skip if it is already started. If you want restart it, please use restart command
  stop      Stop the client
  restart   Restart the client

Install Command Options:
  -n namespace  Namespace where the current ecs is located, default is default
  -m startupMode start up mode, default is manual
  -k license    CHAOS License
  -p applicationInstance The application instance name to which the specified machine belongs，default is chaos-default-app.
  -g applicationGroup The application group to which the specified machine belongs,default is chaos-default-app-group.
  -P port       CHAOS Agent listen port, default is 19527
  -t endpoint   CHAOS endpoint
"
  sendResult 3 "illegal parameters"
}

deploy_chaosblade() {
  local chaosblade_path="/opt/chaosblade"
  if [ ! -e ${DEST_PATH}/chaosblade ]; then
    if [ ! -e ${chaosblade_path} ]; then
        log_warn "chaosblade not exit!"
        sendResult 1 "chaosblade not exit !"
    fi
    return
  fi

  if [ -e ${chaosblade_path} ]; then
    now=$(date "+%Y%m%d%H%M")
    mv ${chaosblade_path} ${chaosblade_path}_${now}
  fi
  mv ${DEST_PATH}/chaosblade /opt
}

create_flags() {
  local start_f="$1"
  local flags=""
  if [ "${start_f}" = ${StartOldFlag} ]; then
    # read flags from file
    flags=$(cat ${DEST_PATH}/${FLAG_FILENAME})
  elif [ "${startupMode}" = ${crontabMode} ]; then
      flags=$(cat ${DEST_PATH}/${FLAG_FILENAME})
  else
    if [ -n "${namespace}" ]; then
      flags="${flags} --namespace ${namespace}"
    fi
    if [ -n "${license}" ]; then
      flags="${flags} --license ${license}"
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
    flags="${flags} --transport.endpoint ${endpoint}"
    # record flag
    echo "${flags}" >${DEST_PATH}/${FLAG_FILENAME} | tr '\n' ' '
  fi
  echo "${flags}" | tr '\n' ' '
}

start0() {
  local start_flag="$1"
  local pid=$(ps -ef | grep chaos/agent | grep -v grep | awk '{print $2}')
  if [ -n "${pid}" ]; then
    echo "0,chaos has been started" | tr '\n' ' '
    return
  fi
  flags=$(create_flags ${start_flag})
  nohup ${DEST_PATH}/${DEST_FILENAME} ${flags} >>${DEST_PATH}/${LOG_FILE_NAME} 2>&1 &
  local count=10
  while :; do
    if [ "${count}" = "0" ]; then
      # if [ -e "${DEST_PATH}/${LOG_FILE}" ];then
      #log_warn `cat ${DEST_PATH}/${LOG_FILE}`
      #fi
      stop >/dev/null 2>&1
      echo "1,start timeout" | tr '\n' ' '
      return
    fi

    if [ -e "${PID_FILE}" ]; then
      if [ $(cat "${PID_FILE}") = "-1" ]; then
        #log_warn "[start] failed:"`cat ${DEST_PATH}/${LOG_FILE}`
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
  log_info "[+starting] chaos is starting..."
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
  local process="chaos/agent"
  pid=$(ps -ef | grep ${process} | grep -v grep | awk '{print $2}')
  if [ -n "${pid}" ]; then
    kill -15 ${pid} >/dev/null 2>&1
    sleep 2
  fi
  pkill -9 -f "${process}"
  rm_rf ${PID_FILE}
  log_info "[+stop] chaos is stopped."
}

# 清除定时任务
clean_crontab() {
  log_info "[-crontab] clean crontab cmd"
  local crontabFile=${crontabPath}/${crontabFileName}
  if [ -f ${crontabFile} ]; then
    rm_rf ${crontabFile}
  fi
}
uninstall() {
  rm_rf ${DEST_PATH}
  clean_crontab
  rm_rf /opt/chaosblade
}
install() {
  # add crontab
  add_crontab
}

uninstall_check() {
   local pid=$(ps -ef | grep chaos/agent | grep -v grep | awk '{print $2}')
#  local pid=$(pgrep -f chaos/agent)
  if [ -n "${pid}" ]; then
    sendResult 1 "uninstall failed"
  else
    sendResult 0 "uninstall successfully"
  fi
}

# 添加定时任务
add_crontab() {
  log_info "[+crontab] add crontab cmd"
  local crontabFile=${crontabPath}/${crontabFileName}
  if [ ! -e ${crontabPath} ]; then
    mk_dir ${crontabPath}
  fi
  if [ ! -f ${crontabFile} ]; then
    touch ${crontabFile}
  fi
  echo "${crontabCmd}" >${crontabFile}
  chmod 644 ${crontabFile}
}

action="$1"
shift

while getopts ":n:k:m:p:g:P:t:" opt; do
  case ${opt} in
  n)
    namespace="${OPTARG}"
    ;;
  k)
    license="${OPTARG}"
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
  \?) ;;

  esac
done

case "${action}" in
install)
  check_license
  stop
  check_root
  install
  deploy_chaosblade
  check_endpoint
  checkChaosBin
  start
  ;;
uninstall)
  stop
  uninstall
  uninstall_check
  ;;
start)
  if [ ! "${startupMode}" = ${crontabMode} ]; then
    deploy_chaosblade
    check_license
    check_root
    check_endpoint
  fi

  checkChaosBin
  start
  ;;
stop)
  stop
  ;;
restart)
  check_license
  stop
  checkChaosBin
  start
  ;;
*)
  usages
  ;;
esac
log_info "[${action}] execute successfully"
exit 0