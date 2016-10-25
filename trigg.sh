#!/bin/bash

set -e

ARGS="${@}"

check_args.sh ${ARGS[@]}
status="$?"

if [[ "$status" != "0" ]]; then
	exit "$status"
fi

declare -A variables=()

trigger_url=""
trigger_token=""
trigger_ref="master"

if [[ "$#" == "0" ]]; then
	echo "!!! No options provided. Exiting..."
	exit 1
fi

echo "start with args:"
echo "${@}"

while true; do
	case "$1" in
		"--trigger-url" )
			shift
			trigger_url="$1"
			shift
		;;

		"--trigger-token" )
			shift
			trigger_token="$1"
			shift
		;;

		"--trigger-ref" )
			shift
			trigger_ref="$1"
			shift
		;;

		"--var" )
			shift
			key="$1"; shift
			value="$1"; shift
			variables+=(["$key"]="$value")
		;;

		* ) shift ;;
	esac

	if [[ "$#" == "0" ]]; then
		break
	fi

done

if [[ "$trigger_url" == "" ]]; then
	echo "!!! The trigger url is no setted. Exiting..."
	exit 1
fi

command_str="curl -X POST"

if [[ "$trigger_token" != "" ]]; then
	command_str="$command_str -F token=$trigger_token"
fi

if [[ "$trigger_ref" != "" ]]; then
	command_str="$command_str -F ref=$trigger_ref"
fi

for key in ${!variables[@]}
do
	value="${variables[$key]}"
	command_str="$command_str -F variables[\"$key\"]=\"$value\""
done

command_str="$command_str $trigger_url"

eval "$command_str"