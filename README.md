# krmfnsops

[![stability-wip](https://img.shields.io/badge/stability-wip-lightgrey.svg)](https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#work-in-progress)

KRM function (Kustomize) to decrypt SOPS encoded resources.

It uses the
[Exec KRM fuctions](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/exec_krm_functions/)
mechanism that is currently cooking both in Kustomize and
[Kpt](https://kpt.dev/).

You use it by configuring a Transformer or Generator:

```yaml
# sops-generator.yaml
apiVersion: kaweezle
# Here by replacing Generator by Transformer it would decrypt existing resources
kind: SecretsGenerator
metadata:
    name: whatever
    annotations:
        # https://kubectl.docs.kubernetes.io/guides/extending_kustomize/#required-alpha-flags
        config.kubernetes.io/function: |
            exec:
              path: ../dist/krmfnsops_linux_amd64/krmfnsops
spec:
    files:
        - ./secret.enc.yaml
```

And configure it in your `kustomization.yaml`:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# When using as a transformer, encrypted resources as specified as usual
# resources:
#  - ./secret.yaml

# transformers:
generators:
    - sops-generator.yaml
```

You need to run `kustomize` with the following flags:

```console
> kustomize build . --enable-alpha-plugins --enable-exec
```

## TODO

-   [ ] Explain integration with Argo CD
-   [ ] Build docker image for both Kpt and inclusion in Argo CD
