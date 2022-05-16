#!/bin/sh

# DEPENDENCEIS
# sops
# kustomize
# age
# yq

#set -uexo pipefail
set -e pipefail

trap "rm -f key.txt *.enc.yaml *.dec.yaml" EXIT SIGINT

age-keygen -o key.txt >/dev/null 2>&1
export SOPS_AGE_KEY_FILE="$(pwd)/key.txt"
export SOPS_AGE_RECIPIENTS=$(grep public key.txt | cut -d' ' -f 4)
sops -e secret.yaml > secret.enc.yaml
kustomize build . --enable-alpha-plugins --enable-exec > secret.dec.yaml
diff <(yq eval -P secret.yaml) <(yq eval -P secret.dec.yaml)
