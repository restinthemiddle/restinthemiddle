#!/bin/ash

set -eo pipefail

# if command starts with an option, prepend mysqld
if [ "${1:0:1}" = '-' ]; then
	set -- restinthemiddle "${@}"
fi

exec "${@}"
