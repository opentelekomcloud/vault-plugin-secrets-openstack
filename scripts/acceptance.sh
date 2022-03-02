#!/usr/bin/env bash
vault server -dev -dev-root-token-id=root -dev-plugin-dir="$VAULT_PLUGIN_DIR" >/dev/null 2>&1 &
pid=$!

go test -count=1 -v -timeout=20m -covermode atomic -coverprofile coverage-func.out -coverpkg ./... ./acceptance
rc=$?

kill $pid
exit $rc
