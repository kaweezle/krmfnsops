# Example: Use an encrypted generator as a source for replacements

In this example, `secrets.yaml` is a KRM (Kubernetes Resource Model) resources
containing all your secrets. You use it as a replacement source for the secrets
in your kustomization.

It contains the following annotations:

```yaml
annotations:
  config.kubernetes.io/function: |
    exec:
      path: ../../krmfnsops
  krmfnsops.kaweezle.com/keep-local-config: "true"
```

The first one makes it going through krmfnsops for decryption. The second one
keeps the resource local to the kustomization and prevents it from being
outputted.

You make it available in the kustomization by adding the following to
`kustomization.yaml`:

```yaml
generators:
  - secrets.yaml
```

And then can use its values for replacement. For instance:

```yaml
replacements:
  - source:
      kind: Secrets
      fieldPath: data.github.password
    targets:
      - select:
          kind: Secret
          name: private-repo
        fieldPaths:
          - stringData.password
```

## Usage

```bash
> cd examples/secrets
# generate the krmfnsops binary
> (cd ../../; go build .)
# Setup AGE keys for encryption/decryption
> mkdir -p ~/.config/sops/age && age-keygen -o ~/.config/sops/age/keys.txt
> export SOPS_AGE_RECIPIENTS=$(cat ~/.config/sops/age/keys.txt | age-keygen -y)
# encrypt secrets.yaml
> sops -e -i secrets.yaml
# test the kustomization
> kustomize build --enable-alpha-plugins --enable-exec

```
