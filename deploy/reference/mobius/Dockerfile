# Image of the Mobius reference stack (deploy/reference/docker-compose.yml).
# The one-shot Mobius build (mobius-build) and both Mobius servers run from it.
#
# Mobius CT_0 Interlude needs Java 27 to build (build.xml checks it) and a full
# JDK at runtime too: the game server compiles its Java scripts when it starts.
FROM eclipse-temurin:27-jdk

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Apache Ant (build.xml needs 1.9.10 or newer). The binary release, checked
# against its published SHA-512. archive.apache.org keeps every version;
# dlcdn.apache.org only the current one, but is faster.
ARG ANT_VERSION=1.10.15

RUN set -eux; \
	apt-get update; \
	apt-get install -y --no-install-recommends ca-certificates curl; \
	rm -rf /var/lib/apt/lists/*; \
	tarball="apache-ant-${ANT_VERSION}-bin.tar.gz"; \
	for base in https://dlcdn.apache.org/ant/binaries https://archive.apache.org/dist/ant/binaries; do \
		if curl -fsSL --retry 3 -o "/tmp/${tarball}" "${base}/${tarball}" \
			&& curl -fsSL --retry 3 -o "/tmp/${tarball}.sha512" "${base}/${tarball}.sha512"; then \
			break; \
		fi; \
		rm -f "/tmp/${tarball}" "/tmp/${tarball}.sha512"; \
	done; \
	test -s "/tmp/${tarball}"; \
	expected="$(tr -d ' \t\r\n' < "/tmp/${tarball}.sha512" | tr 'A-F' 'a-f')"; \
	actual="$(sha512sum "/tmp/${tarball}" | cut -d ' ' -f 1)"; \
	test "${#actual}" -eq 128; \
	case "${expected}" in *"${actual}"*) ;; *) echo "${tarball}: SHA-512 mismatch" >&2; exit 1 ;; esac; \
	mkdir -p /opt/ant; \
	tar -xzf "/tmp/${tarball}" -C /opt/ant --strip-components=1; \
	rm -f "/tmp/${tarball}" "/tmp/${tarball}.sha512"; \
	test -f /opt/ant/lib/ant-launcher.jar

ENV ANT_HOME=/opt/ant

COPY build.sh start.sh /opt/l2go-reference/
RUN chmod 0755 /opt/l2go-reference/build.sh /opt/l2go-reference/start.sh

WORKDIR /srv/mobius
