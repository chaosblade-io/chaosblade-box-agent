#!/usr/bin/env bash
# Copyright 2025 The ChaosBlade Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


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
