#!/bin/bash
# Выкладка сайта go-bpm.tech. На сервере - /opt/go-bpm-site/deploy.sh, вызывается CI по SSH как forced command ключа:
#
#	command="/opt/go-bpm-site/deploy.sh",restrict ssh-ed25519 ... go-bpm-site-ci
#
# CI передаёт в аргументе (SSH_ORIGINAL_COMMAND) коммит сайта, а на stdin - tar.gz собранного сайта (dist/).
# Скрипт распаковывает его в новый релиз, проверяет, переключает симлинк current и оставляет пять последних релизов.
set -euo pipefail

SITE=/var/www/go-bpm.tech
KEEP=5

label=${SSH_ORIGINAL_COMMAND:-manual}
if [[ ! $label =~ ^[A-Za-z0-9._-]{1,64}$ ]]; then
	echo "deploy: неверная метка релиза: $label" >&2
	exit 1
fi

release=$SITE/releases/$(date -u +%Y%m%d-%H%M%S)-$label
mkdir -p "$SITE/releases"
tmp=$(mktemp -d "$SITE/releases/.incoming.XXXXXX")
trap 'rm -rf "$tmp"' EXIT

# не больше 100 МБ: сайт весит пару мегабайт
head -c 100000000 | tar -xz -C "$tmp" --no-same-owner --no-same-permissions

for f in index.html docs/index.html demo/index.html static/style.css; do
	if [[ ! -s $tmp/$f ]]; then
		echo "deploy: в архиве нет $f" >&2
		exit 1
	fi
done
if find "$tmp" -type l | grep -q .; then
	echo "deploy: в архиве есть симлинки" >&2
	exit 1
fi

chmod -R u=rwX,go=rX "$tmp"
mv "$tmp" "$release"
trap - EXIT
ln -sfn "$release" "$SITE/current.new"
mv -T "$SITE/current.new" "$SITE/current"
echo "deploy: выложен $release"

ls -1d "$SITE"/releases/*/ | sort | head -n -$KEEP | xargs -r rm -rf

# nginx отдаёт именно новый релиз: главная совпадает с файлом
if ! curl -sf -H 'Host: go-bpm.tech' http://127.0.0.1/ | cmp -s - "$release/index.html"; then
	echo "deploy: nginx не отдаёт новую главную" >&2
	exit 1
fi
echo "deploy: готово"
