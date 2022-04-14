#!/usr/bin/env bash

# check agent process exists or not
PROCESS_NAME="chaos/agent"
PROCESS_PATH="/opt/chaos"
UPGRADE_LOCK=".lock"
FLAGS_FILE=".flag"
CTL_FILE_PATH="chaosctl.sh"
CRONTAB_MODEL="crontab"
PROCESS_LOG="agent.log"

log() {
    LEVEL=$1
    MSG=$2
    shift
    message="$(date +'%Y-%m-%d %T') [$LEVEL]"
    if [ -f ${PROCESS_PATH}/${PROCESS_LOG} ]; then
        echo -e "${message}" "$@" >> ${PROCESS_PATH}/${PROCESS_LOG}
    fi
    echo -e "${message}" "$@"
}

log_warn() {
    log 'WARN' "$@"
}

log_info() {
    log 'INFO' "$@"
}

check_process() {
    local pid=$(pgrep -f ${PROCESS_NAME})
    if [ -n "${pid}" ]; then
        log_info "Agent has been started"
        exit 0
    fi
}

check_status() {
    if [ ! -e ${PROCESS_PATH} ]; then
        log_warn "process dir not exists"
        exit 1
    fi
    if [ -e ${PROCESS_PATH}/${UPGRADE_LOCK} ]; then
        log_warn "process in upgrading"
        exit 1
    fi
    if [ ! -e ${PROCESS_PATH}/${FLAGS_FILE} ]; then
        log_warn ".flag file not found"
        exit 1
    fi
    if [ ! -e ${PROCESS_PATH}/${CTL_FILE_PATH} ]; then
        log_warn "ctl file not found"
        exit 1
    fi
}

recover() {
    local flags=$(cat ${PROCESS_PATH}/${FLAGS_FILE})
    local ctl=${PROCESS_PATH}/${CTL_FILE_PATH}
    sh ${ctl} start -m ${CRONTAB_MODEL}
    exitCode=$?
    if [ ${exitCode} != 0 ]; then
        log_warn "recover agent process failed, flags: ${flags}"
        exit 2
    fi
    log_info "recover agent process success, flags: ${flags}"
    exit 0
}

check_process
check_status
recover
