#!/bin/sh
# Builds the CGI and the FastCGI program, serves sample/ with both from an
# Apache that the current user starts on 127.0.0.1:8080 (httpd-test.conf), and
# checks that the pages are the same as in hns/testdata, that the files that
# must stay private are refused, and that an idle FastCGI process ends quickly
# on SIGTERM. For FreeBSD with apache24 and ap24-mod_fcgid.
# Usage: sh sample/apache/check.sh   (HTTPD and ROOT can be set in the environment)
set -eu
cd "$(dirname "$0")/../.."
TOP=$(pwd)
ROOT=${ROOT:-$(mktemp -d /tmp/gohns-apache.XXXXXX)}
export ROOT
HTTPD=${HTTPD:-/usr/local/sbin/httpd}
CONF=$TOP/sample/apache/httpd-test.conf
URL=http://127.0.0.1:8080

mkdir -p "$ROOT/logs" "$ROOT/run/fcgid"
for kind in cgi fcgi; do
	d=$ROOT/htdocs/$kind
	mkdir -p "$d"
	tar -C sample -cf - config.json ice diary | tar -C "$d" -xf -
	sed "s/index.fcgi/index.$kind/" sample/.htaccess > "$d/.htaccess"
	go build -o "$d/index.$kind" "./cmd/${kind}hns"
done

trap '"$HTTPD" -f "$CONF" -k stop >/dev/null 2>&1 || true' EXIT
"$HTTPD" -f "$CONF" -k start
i=0
until curl -s -o /dev/null "$URL/cgi/"; do
	i=$((i + 1))
	[ $i -le 50 ] || { echo "httpd did not start; see $ROOT/logs/error.log"; exit 1; }
	sleep 0.1
done

fail=0
norm() { sed 's|<hr>[^<]*<br>|<hr>TIME<br>|'; }
check() { # name of the file in hns/testdata, query
	norm < "$TOP/hns/testdata/$1.html" > "$ROOT/want"
	for kind in cgi fcgi; do
		curl -s "$URL/$kind/$2" | norm > "$ROOT/got"
		cmp -s "$ROOT/want" "$ROOT/got" || { echo "FAIL  $kind/$2 differs from hns/testdata/$1.html"; fail=1; }
	done
}
status() { # wanted status, path
	got=$(curl -s -o /dev/null -w '%{http_code}' "$URL$2")
	[ "$got" = "$1" ] || { echo "FAIL  GET $2 is $got, not $1"; fail=1; }
}

# keep in step with the cases in hns/hns_test.go
check recent ""
check monthday_0101 "?0101"
check month_202403 "?202403"
check month_202403a "?202403a"
check month_202403b "?202403b"
check month_202403c "?202403c"
check day_20240301 "?20240301"
check year_2023 "?2023"
check japanese_202405 "?202405"
check errors_202406 "?202406"
check title "?title"
check title_cat_tech "?title&CAT=tech"
check recent_cat_diary "?CAT=diary"

status 200 /fcgi/ice/style.css
for kind in cgi fcgi; do
	status 403 /$kind/config.json
	status 403 /$kind/diary/2024/d20240101.hnf
done

# an idle FastCGI process must not wait for requests that are not there
pid=$(pgrep -f "$ROOT/htdocs/fcgi/index.fcgi" | head -1)
kill -TERM "$pid"
i=0
while [ $i -lt 40 ]; do
	s=$(ps -o state= -p "$pid" 2>/dev/null || true)
	case "$s" in "" | Z*) break ;; esac
	sleep 0.1
	i=$((i + 1))
done
[ $i -lt 12 ] || { echo "FAIL  an idle FastCGI process needed $((i * 100)) ms or more to end on SIGTERM"; fail=1; }

[ $fail = 0 ] && echo "all checks passed (files in $ROOT)"
exit $fail
