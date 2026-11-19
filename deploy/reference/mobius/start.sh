#!/bin/bash
# Starts a Mobius server of the reference stack: start.sh login|game.
#
# 1. config/ is rebuilt at every start: Mobius's defaults (config.default/,
#    kept by build.sh), then our overrides from deploy/reference/config/<server>/,
#    mounted read-only at /opt/l2go-reference/overrides:
#    - <Name>.ini holds "Key = Value" lines. Each key replaces the line that
#      sets it in Mobius's config/<Name>.ini, or is appended (with a warning:
#      check the key name) when Mobius's file has none.
#    - Any other file (ipconfig.xml) is copied into config/ as is.
# 2. Mobius's start command runs in the server directory, as in its
#    LoginServerTask.sh and GameServerTask.sh:
#      java $(cat java.cfg) -jar ../libs/<Login|Game>Server.jar
#    MOBIUS_JAVA_OPTS, when set, replaces java.cfg. The output goes to the
#    container log (not log/stdout.log); Mobius's own log files are in log/.
set -euo pipefail

server="${1:-}"
case "$server" in
	login) jar=LoginServer.jar ;;
	game) jar=GameServer.jar ;;
	*)
		echo "usage: start.sh login|game" >&2
		exit 2
		;;
esac

root=/srv/mobius
home="$root/$server"
overrides=/opt/l2go-reference/overrides

log() { echo "start.sh ($server): $*"; }
fail() {
	echo "start.sh ($server): $*" >&2
	exit 1
}

# set_key FILE KEY VALUE sets KEY in the Java properties file FILE: every line
# that sets KEY is replaced by "KEY = VALUE"; the line is appended when none
# does, and then it returns 3.
set_key() {
	local file="$1" status=0
	OVERRIDE_KEY="$2" OVERRIDE_VALUE="$3" awk '
		BEGIN { key = ENVIRON["OVERRIDE_KEY"]; value = ENVIRON["OVERRIDE_VALUE"]; found = 0 }
		{
			# A properties key runs up to the first blank, "=" or ":".
			line = $0
			sub(/^[ \t]+/, "", line)
			name = line
			sub(/[ \t=:].*/, "", name)
			first = substr(line, 1, 1)
			if (first != "#" && first != "!" && name == key) {
				print key " = " value
				found = 1
				next
			}
			print
		}
		END {
			if (!found) {
				print key " = " value
				exit 3
			}
		}
	' "$file" > "$file.tmp" || status=$?
	if [ "$status" -ne 0 ] && [ "$status" -ne 3 ]; then
		rm -f "$file.tmp"
		fail "could not set $2 in $file"
	fi
	mv "$file.tmp" "$file" || fail "could not write $file"
	return "$status"
}

# apply_ini OVERRIDE applies an override file to the config file of the same name.
apply_ini() {
	local override="$1" name target line key value status
	local skip_re='^[[:space:]]*([#!].*)?$'
	local pair_re='^[[:space:]]*([^[:space:]=:]+)[[:space:]]*[=:][[:space:]]*(.*)$'
	name="$(basename "$override")"
	target="$home/config/$name"
	[ -f "$target" ] || fail "$name: Mobius has no config/$name to override"
	while IFS= read -r line || [ -n "$line" ]; do
		line="${line%$'\r'}"
		[[ $line =~ $skip_re ]] && continue
		[[ $line =~ $pair_re ]] || fail "$name: not a Key = Value line: $line"
		key="${BASH_REMATCH[1]}"
		value="${BASH_REMATCH[2]}"
		value="${value%"${value##*[![:space:]]}"}" # trailing blanks
		status=0
		set_key "$target" "$key" "$value" || status=$?
		if [ "$status" -eq 3 ]; then
			log "config/$name: $key = $value (WARNING: appended, Mobius's $name has no $key)"
		else
			log "config/$name: $key = $value"
		fi
	done < "$override"
}

if [ ! -f "$root/libs/$jar" ] || [ ! -d "$home/config.default" ]; then
	fail "no Mobius build in $root; run make reference-build"
fi
if [ ! -f "$overrides/Database.ini" ]; then
	fail "no overrides in $overrides (deploy/reference/config/$server/ is not mounted)"
fi

log "config: Mobius defaults plus deploy/reference/config/$server/"
rm -rf "$home/config"
cp -R "$home/config.default" "$home/config"
for override in "$overrides"/*; do
	[ -f "$override" ] || continue
	case "$override" in
		*.ini) apply_ini "$override" ;;
		*)
			cp "$override" "$home/config/"
			log "config/$(basename "$override"): replaced by ours"
			;;
	esac
done

mkdir -p "$home/log"
cd "$home"
java_opts="${MOBIUS_JAVA_OPTS:-$(tr -d '\r' < java.cfg)}"
log "java $java_opts -jar ../libs/$jar"
# Unquoted on purpose: one argument per option, like Mobius's start scripts.
set -f
# shellcheck disable=SC2086
exec java $java_opts -jar "../libs/$jar"
