# krmfnsops

[![stability-beta](https://img.shields.io/badge/stability-beta-33bbff.svg)](https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#beta)

krmfnsops is a
[kustomize plugin](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/)
that you can use to decrypt resources encrypted with
[SOPS](https://github.com/mozilla/sops). It uses the
[Exec KRM functions](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/exec_krm_functions/)
mechanism that is currently cooking both in Kustomize and
[Kpt](https://kpt.dev/).

As it embeds SOPS, you **don't need** to install SOPS in addition to krmfnsops.

You can use it either as a Generator or as a Transformer (see below). To obtain
the expected results, you need to run `kustomize` with the following flags:

```console
> kustomize build . --enable-alpha-plugins --enable-exec
```

## Use case 1/4: Configuration as a Generator

You create a `sops-generator.yaml` resource for the generator:

```yaml
# sops-generator.yaml
apiVersion: kaweezle
# suffix Generator
kind: SecretsGenerator
metadata:
  name: whatever
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: ../dist/krmfnsops_linux_amd64/krmfnsops
spec:
  files:
    - ./secret.enc.yaml
```

The files to decrypt are specified in `spec/files`. Then reference the generator
in the `kustomization.yaml` configuration file:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

generators:
  - sops-generator.yaml
```

## Use case 2/4: Configuration as a Transformer

**CAUTION** Sops computes a Message authentication code from the source file and
checks it after decrypt in order to verify that the encrypted file has not been
modified. However, the transformer doesn't receive the original source, but an
object representing each resource inside it, modified by kustomize for
processing purposes. In consequence, the MAC verification is **disabled** in
transformer mode.

The following is the configuration for the function in Transformer mode:

```yaml
# sops-transformer.yaml
apiVersion: kaweezle
# suffix Transformer
kind: SecretsTransformer
metadata:
  name: whatever
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: ../dist/krmfnsops_linux_amd64/krmfnsops

# Note that there is no spec
```

And configure it in your `kustomization.yaml`:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# Add the encrypted resources
resources:
  - ./secret.yaml

transformers:
  - sops-transformer.yaml
```

## Use case 3/4: Configuration as an _In place_ Generator

In this use case, the generator configuration is the actual resource that needs
to be added. Let's imagine that you have this secret in your kustomization:

```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  labels:
    argocd.argoproj.io/secret-type: repository
  name: private-repo
  namespace: argocd
stringData:
  password: my-password
  type: git
  url: https://github.com/argoproj/private-repo
  username: my-username
```

You want it encrypted. For that, you add the krmfnsops function annotation:

```yaml
annotations:
  config.kubernetes.io/function: |
    exec:
      path: krmfnsops
```

and encrypt it with sops:

```console
> sops -e -i secret.yaml
```

You obtain an encrypted version of the secret that can be added _as is_ as a
generator in your kustomization:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

generators:
  - secret.yaml
```

The command:

```console
> kustomize build --enable-alpha-plugins --enable-exec .
```

will output the unencrypted secret.

## Use case 4/4: Use an encrypted generator as a source for replacements

In this use case, we use an encrypted generator that contains all our secrets:

```yaml
# secrets.yaml
apiVersion: krmfnsops.kaweezle.com/v1alpha1
kind: Secrets
metadata:
  name: all-my-secrets
  annotations:
    # this annotation will keep the resource out of the output
    krmfnsops.kaweezle.com/keep-local-config: "true"
    # this annotation will perform decryption for us
    config.kubernetes.io/function: |
      exec:
        path: ../../krmfnsops
data:
  github:
    password: gh_<github_token>
    application_secret: <secret>
  ovh:
    consumer_key: <secret>
    application_secret: <secret>
```

Note that it contains the function annotation, and a new annotation
`krmfnsops.kaweezle.com/keep-local-config`. This annotation will make the
resource available in the kustomization pipeline but will keep it out of the
output. This will allow us to use the data of the resource as a source for
replacements.

We encrypt our secrets with the following command:

```console
> sops -e -i secret.yaml
```

Now we can have a secret in our kustomization with a fake password:

```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  labels:
    argocd.argoproj.io/secret-type: repository
  name: private-repo
stringData:
  password: this-is-a-fake-password
  type: git
  url: https://github.com/argoproj/private-repo
  username: my-username
```

A make the kustomization replace the fake password with the unencrypted one:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# Add the faked resource
resources:
  - secret.yaml

# Add our encypted secrets
generators:
  - secrets.yaml

# Replace the fake password with te real one
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

Now the kustomization gives:

```console
❯ kustomize build --enable-alpha-plugins --enable-exec
apiVersion: v1
kind: Secret
metadata:
  labels:
    argocd.argoproj.io/secret-type: repository
  name: private-repo
stringData:
  password: gh_<github_token>
  type: git
  url: https://github.com/argoproj/private-repo
  username: my-username
```

The files are available in `examples/secrets`.

## Installation

With each [Release](https://github.com/kaweezle/krmfnsops/releases), we provide
binaries for most platforms as well as Alpine based packages. Typically, you
would install it on linux with the following command:

```console
> KRMFNSOPS_VERSION="v0.1.5"
> curl -sLo /usr/local/bin/krmfnsops https://github.com/kaweezle/krmfnsops/releases/download/${KRMFNSOPS_VERSION}/krmfnsops_${KRMFNSOPS_VERSION}_linux_amd64
```

## Argo CD integration

To use krmfnsops with Argo CD, you need to:

- Make the `krmfnsops`binary available to the `argo-repo-server` pod.
- Have Argo CD run kustomize with the `--enable-alpha-plugins --enable-exec`
  parameters.
- Make the decrypting keys available to `krmfnsops`.

To add krmfnsops on argo-repo-server, the
[Argo CD documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/custom_tools/)
provides different methods to make custom tools available.

If you get serious about Argo CD, you will probably end up cooking your own
image. This
[docker file](https://github.com/antoinemartin/autocloud/blob/deploy/citest/repo-server/Dockerfile#L45)
shows how to use the above installation instructions in your image. To
summarize:

```Dockerfile
FROM argoproj/argocd:latest

ARG KRMFNSOPS_VERSION=v0.1.5

# Switch to root for the ability to perform install
USER root

# Install tools
RUN apt-get update && \
    apt-get install -y curl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    curl -sLo /usr/local/bin/krmfnsops https://github.com/kaweezle/krmfnsops/releases/download/${KRMFNSOPS_VERSION}/krmfnsops_${KRMFNSOPS_VERSION}_linux_amd64

USER argocd
```

For the other points, we assume in the following that your Argo CD deployment
occurs through kustomize. Here is the kustomization file layout:

```console
.
├── argocd-cm.yaml
├── argocd-repo-server-patch.yaml
├── kustomization.yaml
├── secrets.yaml
└── sops-generator.yaml
```

The base `kustomization.yaml` contains:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: argocd

resources:
  # The standard Argo CD installation
  - https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

generators:
  # This generator will generate the secret containing our AGE key
  - sops-generator.yaml

# Kustomization of the Argo CD standard installation
patches:
  - path: argocd-repo-server-patch.yaml
    target:
      kind: Deployment
      name: argocd-repo-server
  - path: argocd-cm.yaml
```

The `argocd-cm.yaml` patch contains the configuration needed for the parameters:

```yaml
# argocd-cm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
data:
  # Options to enable exec plugins (krmfnsops).
  kustomize.buildOptions: "--enable-alpha-plugins --enable-exec"
  ...
```

The `sops-generator.yaml` file will allow decrypting our secrets on deployment:

```yaml
# sops-generator.yaml
apiVersion: iknite.krm.kaweezle.com/v1beta1
kind: SopsGenerator
metadata:
  name: secrets
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: krmfnsops
spec:
  files:
    - ./secrets.yaml
```

The `secrets.yaml` file is a SOPS encrypted file containing all the secrets
needed by Argo CD, including the key used by krmfnsops on the server.

We will use here an [Age](https://github.com/FiloSottile/age) key as an example.
The key is generated and exported as a base64 payload with the following:

```console
> mkdir -p ~/.config/sops/age && age-keygen -o ~/.config/sops/age/keys.txt
> cat ~/.config/sops/age/keys.txt | openssl base64 -e -A
<base64 encoded key>
```

Then it's added it to the `secrets.yaml` file:

```yaml
# secrets.yaml
# In addition, this file would contain:
# - The git credentials to access private repositories
# - The admin password
# - The external OIDC identification credentials (client secret, ...)
# ...
---
apiVersion: v1
kind: Secret
metadata:
  name: argocd-sops-private-keys
type: Opaque
data:
  keys.txt: <base64 encoded key>
```

It is encrypted with the following command:

```console
> export SOPS_AGE_RECIPIENTS=$(cat ~/.config/sops/age/keys.txt | age-keygen -y)
> sops -e -i secrets.yaml
```

The file now contains encrypted entries:

```yaml
apiVersion: v1
kind: Secret
metadata:
    name: argocd-sops-private-keys
type: Opaque
data:
    age_key.txt: ENC[AES256_GCM,data:xbP4U...,type:str]
sops:
    age:
        - recipient: age1...
          enc: | ...
    kms: []
```

⚠️ To keep the encrypted entries to a minimum, add a `.sops.yaml` file to your
project with the following:

```yaml
creation_rules:
  - encrypted_regex: "^(data|stringData)$"
    # You can put your age key here (obtain it with cat ~/.config/sops/age/keys.txt| age-keygen -y)
    # age: age1..
```

Now that the secret is configured, making it available for the argocd-repo-sever
is done with the `argocd-repo-server-patch.yaml` patch file:

```yaml
# argocd-repo-server-patch.yaml
# Use custom image
- op: replace
  path: /spec/template/spec/containers/0/image
  value: <your custom image>
# Add sops secrets volume
- op: add
  path: /spec/template/spec/volumes/-
  value:
    name: argocd-sops-private-keys
    secret:
      secretName: argocd-sops-private-keys
      optional: true
      defaultMode: 420
# Mount volume on server
- op: add
  path: /spec/template/spec/containers/0/volumeMounts/-
  value:
    mountPath: /home/argocd/.config/sops/age
    name: argocd-sops-private-keys
```

Deploy Argo CD with:

```console
> kustomize build --enable-alpha-plugins --enable-exec . | kubectl apply -f
```

## Similar projects

- [viaduct-ai/kustomize-sops](https://github.com/viaduct-ai/kustomize-sops)
- [goabout/kustomize-sopssecretgenerator](https://github.com/goabout/kustomize-sopssecretgenerator)
  that also contains a more complete list of
  [other alternatives](https://github.com/goabout/kustomize-sopssecretgenerator#alternatives).

```

```
