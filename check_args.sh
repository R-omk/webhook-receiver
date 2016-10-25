#!/bin/bash


while true; do
	case "$1" in

		"--eq" )
			shift
			what="$1"; shift
			with="$1"; shift
			if [[ "$what" != "$with" ]]; then
				printf "!!! The condintion '%s == %s' is not satisfied. Exiting ...\n" \
					"$what" "$with"
				exit 1
			fi
		;;

		"--match" )
			shift
			preg="$1"; shift
			string="$1"; shift
			echo "$string" | egrep "$preg" >/dev/null || {
				printf "!!! The condintion '%s match %s' is not satisfied. Exiting ...\n" \
					"$string" "$preg"
				exit 1
			}
		;;

		* ) shift ;;
	esac
	
	if [[ "$#" == "0" ]]; then
		break
	fi
done

exit 0