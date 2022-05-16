#!/usr/bin/env sh

set -e

echo "::group::Setup"

cd $GITHUB_WORKSPACE

if [[ -z "$INPUT_APK_FILES" ]]; then
    echo "No APK files given!"
    exit 1
fi

if  [[ -z "$INPUT_SIGNATURE_KEY_NAME" ]]; then
    echo "No signature key name given!"
    exit 1
fi

if  [[ -z "$INPUT_SIGNATURE_KEY" ]]; then
    echo "No signature key given!"
    exit 1
else
    signature_file="/root/$INPUT_SIGNATURE_KEY_NAME"
    printf "$INPUT_SIGNATURE_KEY" > "$signature_file"
fi

if [[ -z "$INPUT_DESTINATION" ]]; then
    echo "No destination given!"
    exit 1
fi

files=$(ls -1 $INPUT_APK_FILES 2>/dev/null)
files_count=$(echo "$files" | wc -l)
if [[ "$files_count" -eq 0 ]]; then
    echo "There are no apk files matching $INPUT_APK_FILES"
    exit 1
fi

archs=$(ls -1 $INPUT_APK_FILES 2>/dev/null | sed -E 's/^.*\.(.*)\.apk$/\1/g')
archs_count=$(echo "$archs" | wc -l)
if [[ "$archs_count" -eq 0 ]]; then
    echo "No architectures found in APK files: $files"
    exit 1
fi

echo "Creating repo in $INPUT_DESTINATION from $files_count APK files with $archs_count architectures..."

echo "::endgroup::"


echo "::group::Creating repo in $INPUT_DESTINATION"


rm -rf $INPUT_DESTINATION
mkdir -p $INPUT_DESTINATION

for arch in $archs; do
    arch_directory="${INPUT_DESTINATION}/$arch"
    echo "Creating directory $arch_directory"
    mkdir -p "$arch_directory"
done

for file in $(echo "$files"); do
    file_basename=$(basename "$file")
    file_arch=$(echo "$file_basename" | sed -E 's/^.*\.(.*)\.apk$/\1/g')
    destination_filename=$(echo "$file_basename" | sed -E 's/^(.*)\.[^.]*\.apk$/\1.apk/g')
    echo "Copying ${file} to ${INPUT_DESTINATION}/${file_arch}/${destination_filename}..."
    cp -f "${file}" "${INPUT_DESTINATION}/${file_arch}/${destination_filename}"
done

echo "::endgroup::"


echo "::group::Creating indexes"

for arch in $archs; do
    arch_directory="${INPUT_DESTINATION}/$arch"
    index_file="${arch_directory}/APKINDEX.tar.gz"
    apk index -o "${index_file}" "${arch_directory}"/*.apk 2>/dev/null
    abuild-sign -k "${signature_file}" "${index_file}"
done

echo "::endgroup::"


