#!/bin/sh
# Entrypoint for the integration-test container: starts the local Node.js
# distribution fixture so the test suite never touches the real network.
set -eu

fixture-server -addr 127.0.0.1:9123 &

until curl -fsS http://127.0.0.1:9123/index.json > /dev/null 2>&1; do
    sleep 0.1
done

export DRIFTR_NODE_MIRROR=http://127.0.0.1:9123
exec /home/driftr/test.sh
