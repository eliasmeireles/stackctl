#!/usr/bin/env bash

log_info() {
  echo "[INFO] $*"
}

log_warn() {
  echo "[WARN] $*"
}

log_error() {
  echo "[ERROR] $*"
}

log_ok() {
  echo "[OK] $*"
}

log_skip() {
  echo "[SKIP] $*"
}

check_instance_running() {
  local instance_name=$1
  if ! multipass info "${instance_name}" >/dev/null 2>&1; then
    log_error "Instance '${instance_name}' does not exist"
    return 1
  fi
  
  local state=$(multipass info "${instance_name}" | grep State | awk '{print $2}')
  if [ "${state}" != "Running" ]; then
    log_error "Instance '${instance_name}' is not running (state: ${state})"
    return 1
  fi
  
  return 0
}

exec_in_instance() {
  local instance_name=$1
  shift
  multipass exec "${instance_name}" -- "$@"
}

get_instance_ip() {
  local instance_name=$1
  multipass info "${instance_name}" | grep IPv4 | awk '{print $2}'
}
