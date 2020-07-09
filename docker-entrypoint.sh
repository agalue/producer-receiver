#!/bin/bash -e

function join { local IFS="$1"; shift; echo "$*"; }

IFS=$'\n'
OPTIONS=("acks=1")
for VAR in $(env)
do
  env_var=$(echo "$VAR" | cut -d= -f1)
  if [[ $env_var =~ ^KAFKA_ ]]; then
    key=$(echo "$env_var" | cut -d_ -f2- | tr '[:upper:]' '[:lower:]' | tr _ .)
    val=${!env_var}
    if [[ $key == "manager."* ]]; then
      echo "[Skipping] '$key'"
    else
      echo "[Configuring] '$key'='$val'"
      OPTIONS+=("$key=$val")
    fi
  fi
done

exec /producer-receiver \
  -bootstrap "${BOOTSTRAP_SERVERS}" \
  -topic "${TOPIC-events}" \
  -group-id "${GROUP_ID-producer-go-client}" \
  -message-kind "${MESSAGE_KIND-event}" \
  -parameters "$(join , ${OPTIONS[@]})" \
