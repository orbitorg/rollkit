#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")"

CM_VERSION="v1.0.0-alpha.2"
CM_PROTO_URL=https://raw.githubusercontent.com/cometbft/cometbft/$CM_VERSION/proto/cometbft

CM_PROTO_FILES=(
  abci/v1/types.proto
  version/v1/types.proto
  types/v1/types.proto
  types/v1/evidence.proto
  types/v1/params.proto
  types/v1/validator.proto
  state/v1/types.proto
  crypto/v1/proof.proto
  crypto/v1/keys.proto
  libs/bits/v1/types.proto
  p2p/v1/types.proto
)

echo Fetching protobuf dependencies from CometBFT $CM_VERSION
for FILE in "${CM_PROTO_FILES[@]}"; do
  echo Fetching "$FILE"
  mkdir -p "cometbft/$(dirname $FILE)"
  curl -sSL "$CM_PROTO_URL/$FILE" > "cometbft/$FILE"
done
