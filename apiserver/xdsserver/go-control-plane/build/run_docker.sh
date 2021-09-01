#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

IMAGE_HUB="gcr.io/istio-testing/go-control-plane-ci"
IMAGE_TAG=$(grep "${IMAGE_HUB}" .circleci/config.yml | sed -e "s#.*${IMAGE_HUB}:\(.*\)#\1#" | uniq)

if [ -z "${IMAGE_TAG}" ]; then
  echo "failed to extract the image tag for ${IMAGE_HUB}"
  exit 1
fi

docker run -v $(pwd):/go-control-plane "${IMAGE_HUB}":"${IMAGE_TAG}" $*
