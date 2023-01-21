#!/bin/bash

# DEPENDENCEIS
# sops
# kustomize
# age
# yq

#set -uexo pipefail
set -e pipefail

trap "rm -f key.txt *.enc.yaml *.dec.yaml" EXIT

echo "Generating AGE key in key.txt..."
age-keygen -o key.txt >/dev/null 2>&1
export SOPS_AGE_KEY_FILE="$(pwd)/key.txt"
export SOPS_AGE_RECIPIENTS=$(grep public key.txt | cut -d' ' -f 4)
echo "Encrypting secret.yaml -> secret.enc.yaml with key.txt..."
sops -e secret.yaml > secret.enc.yaml
echo "Encrypting secret2.yaml -> secret2.enc.yaml with key.txt..."
sops -e secret2.yaml > secret2.enc.yaml
echo "Encrypting secret3.yaml -> secret3.enc.yaml with key.txt..."
sops -e secret3.yaml > secret3.enc.yaml
echo "Running kustomize with transformer..."
kustomize build . --enable-alpha-plugins --enable-exec > secret.dec.yaml
cat secret.yaml > expected.dec.yaml
echo "---" >> expected.dec.yaml
cat secret3.exp.yaml >> expected.dec.yaml
echo "---" >> expected.dec.yaml
cat secret2.yaml >> expected.dec.yaml
diff <(yq eval -P expected.dec.yaml) <(yq eval -P secret.dec.yaml)
echo "Secret has been decoded ok 🎉"
