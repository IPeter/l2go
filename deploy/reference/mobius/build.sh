#!/bin/bash
# One-shot Mobius build of the reference stack (the mobius-build service).
#
# Compiles the Mobius checkout, mounted read-only at /mobius, with Mobius's own
# Ant build (build.xml, target "jar"), then lays the server out in the
# mobius-server volume like Mobius's release zip:
#
#   /srv/mobius/libs/   LoginServer.jar, GameServer.jar and dist/libs (minus -sources jars)
#   /srv/mobius/login/  dist/login, with config/ kept as config.default/
#   /srv/mobius/game/   dist/game, with config/ kept as config.default/
#
# start.sh rebuilds config/ from config.default/ and our overrides at every
# server start.
#
# It builds only when the volume has no complete build, or when
# MOBIUS_FORCE_BUILD=1 (make reference-build). Otherwise it says whether the
# checkout changed since the last build, and exits.
set -euo pipefail

src=/mobius
out=/srv/mobius
ant_build=/tmp/mobius-ant # Ant's ${build}: bin/ (classes) and dist/ (jars)
stamp="$out/.build-stamp" # the checkout it was built from; mtime = build start
source_name="${MOBIUS_SOURCE:-$src}"

log() { echo "mobius-build: $*"; }
fail() {
	echo "mobius-build: $*" >&2
	exit 1
}

if [ ! -f "$src/build.xml" ] || [ ! -d "$src/java" ] || [ ! -d "$src/dist/login" ] || [ ! -d "$src/dist/game" ]; then
	fail "$source_name is not a Mobius CT_0 Interlude checkout (build.xml, java/, dist/); set MOBIUS_DIR"
fi

if [ "${MOBIUS_FORCE_BUILD:-0}" != 1 ] && [ -f "$stamp" ] \
	&& [ -f "$out/libs/LoginServer.jar" ] && [ -f "$out/libs/GameServer.jar" ] \
	&& [ -d "$out/login/config.default" ] && [ -d "$out/game/config.default" ]; then
	built_from="$(cat "$stamp")"
	changed="$(find "$src/build.xml" "$src/java" "$src/dist" -newer "$stamp" -print -quit 2> /dev/null || true)"
	if [ "$built_from" != "$source_name" ]; then
		log "the build in the volume is from $built_from, not $source_name; run make reference-build to rebuild"
	elif [ -n "$changed" ]; then
		log "$source_name changed since the last build (e.g. $changed); run make reference-build to rebuild"
	else
		log "the build is up to date with $source_name"
	fi
	exit 0
fi

# The stamp is taken before compiling, so files changed during the build count
# as changes next time. It replaces the old stamp only when the build is done.
mkdir -p "$out"
echo "$source_name" > "$out/.build-stamp.next"

log "building $source_name"
java -version
rm -rf "$ant_build"

# Ant's launcher class, as in Ant's documented "java ... Launcher" form (no
# bin/ant script). -Dbuild moves Ant's output out of the read-only checkout:
# build.xml writes to ../build by default.
ant=(java -Dant.home="$ANT_HOME" -cp "$ANT_HOME/lib/ant-launcher.jar" org.apache.tools.ant.launch.Launcher -noinput)
"${ant[@]}" -version
"${ant[@]}" -buildfile "$src/build.xml" "-Dbuild=$ant_build" jar

log "laying out the server in $out"
stage="$out/.stage"
rm -rf "$stage"
mkdir -p "$stage/libs"
cp "$ant_build/dist/libs/LoginServer.jar" "$ant_build/dist/libs/GameServer.jar" "$stage/libs/"
# The jar manifests' Class-Path points at ../libs/<jar> for these.
find "$src/dist/libs" -maxdepth 1 -type f -name '*.jar' ! -name '*-sources.jar' -exec cp {} "$stage/libs/" ';'
for server in login game; do
	cp -R "$src/dist/$server" "$stage/$server"
	mv "$stage/$server/config" "$stage/$server/config.default"
done

rm -rf "$out/libs" "$out/login" "$out/game"
mv "$stage/libs" "$stage/login" "$stage/game" "$out/"
rmdir "$stage"
mv "$out/.build-stamp.next" "$stamp"
rm -rf "$ant_build"
log "done: $(find "$out/libs" -name '*.jar' | wc -l) jars in $out/libs, servers in $out/login and $out/game"
