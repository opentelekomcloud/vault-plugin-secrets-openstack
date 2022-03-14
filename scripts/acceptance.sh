#!/usr/bin/env bash
vault server -dev -log-level=debug -dev-root-token-id=root -dev-plugin-dir="$VAULT_PLUGIN_DIR" >./output.log 2>&1 &
pid=$!

go test -count=1 -v -timeout=20m -tags=acceptance -parallel=1 ./acceptance
rc=$?

kill $pid
exit $rc
