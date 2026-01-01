#!/bin/sh
# Extract the trailing number from the Pod Name (e.g., bigtablelite-0 -> 0)
ORDINAL=${HOSTNAME##*-}
exec /app/shard_server --shard-id=$ORDINAL