# Make Alpine Linux APK Package Index

This action builds an Alpine Linux package repository from a set of APK files.

## Inputs

## `apk_files`

**Required** The files to include in the repo. Default `"dist/*.apk"`.

## `signature_key`

**Required** The RSA private key to sign the repository.

## `signature_key_name`

**Required** The signature key name. It needs to match the name of the public
key that is installed in `/etc/apk/keys`. For instance, if the public key
filename is `kaweezle-devel@kaweezle.com-c9d89864.rsa.pub`, the name of the key
should be `kaweezle-devel@kaweezle.com-c9d89864.rsa`.

## `destination`

**Required** The directory where to create the repo. Default `"dist/repo"`.

## Outputs

None.

## APK file names

The action expects the APK file name to have the following syntax:

```
<package_name>-<version>.<arch>.apk
```

For instance:

```
iknite-0.1.8.x86_64.apk
```

## Example usage

```yaml
- name: Build APK repo
  uses: ./.github/actions/make-apkindex
  with:
      apk_files: dist/*.apk
      signature_key: "${{ secrets.GPG_PRIVATE_KEY }}"
      signature_key_name: kaweezle-devel@kaweezle.com-c9d89864.rsa
      destination: dist/repo
```
