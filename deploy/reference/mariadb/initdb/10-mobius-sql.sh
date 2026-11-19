#!/bin/bash
# Loads Mobius's SQL into the reference database. The mariadb image runs the
# scripts in /docker-entrypoint-initdb.d once, on the first start with an empty
# data volume, so after a database reset the next start loads it again.
#
# Mobius keeps the login and the game tables in one database: both
# Database.ini URLs name l2jmobiusinterlude (MARIADB_DATABASE). As in Mobius's
# DatabaseInstaller, the login scripts run first, then the game scripts, each
# folder in file name order. The folders are MOBIUS_DIR/dist/db_installer/sql/
# login and .../game, mounted read-only at /mobius-sql by docker-compose.yml.
set -eo pipefail

mobius_sql() {
	if declare -F docker_process_sql > /dev/null; then
		# Sourced by the image's entrypoint (the usual case: this file is not
		# executable). Its client runs as root in $MARIADB_DATABASE.
		docker_process_sql
	else
		# Run as a program: the entrypoint's temporary server, over its socket.
		MYSQL_PWD="$MARIADB_ROOT_PASSWORD" mariadb --protocol=socket -uroot --database="$MARIADB_DATABASE"
	fi
}

for mobius_sql_dir in /mobius-sql/login /mobius-sql/game; do
	if [ ! -d "$mobius_sql_dir" ]; then
		echo "mobius-sql: $mobius_sql_dir is missing; is MOBIUS_DIR a Mobius CT_0 Interlude checkout?" >&2
		exit 1
	fi
	# Mobius's file names are lower case, so byte order is the installer's
	# case-insensitive order.
	mobius_sql_files="$(find "$mobius_sql_dir" -maxdepth 1 -type f -name '*.sql' | LC_ALL=C sort)"
	if [ -z "$mobius_sql_files" ]; then
		echo "mobius-sql: no .sql files in $mobius_sql_dir" >&2
		exit 1
	fi
	while IFS= read -r mobius_sql_file; do
		echo "mobius-sql: ${mobius_sql_file#/mobius-sql/}"
		mobius_sql < "$mobius_sql_file"
	done <<< "$mobius_sql_files"
done
echo "mobius-sql: Mobius login and game tables loaded into $MARIADB_DATABASE"
