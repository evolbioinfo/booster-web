#!/usr/bin/env bash

sleep 30  # Simple workaround for a race-condition between condor binary and firewall rules creation

cat /home/booster/booster-web.toml.template | envsubst > /home/booster/booster-web.toml

cat /home/booster/booster-web.toml >&2

/home/booster/booster-web --config /home/booster/booster-web.toml &

nginx -g 'daemon off;';
