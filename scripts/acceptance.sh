vault server -dev -dev-root-token-id=root -dev-plugin-dir="$VAULT_PLUGIN_DIR" > /dev/null 2>&1 &

rc=0
go test -v -count 1 -timeout 20m -covermode atomic -coverprofile coverage-func.out ./acceptance/... || rc=1

pkill vault

exit $rc