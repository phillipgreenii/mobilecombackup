#!/bin/sh

set -e

echo "Build Project"
devbox run build-cli

echo "Create Repo Directory"
repodir=./tmp/full-test
rm -rf "$repodir"
mkdir -p $repodir

echo "Initialize Repository"
./mobilecombackup init --repo-root="$repodir"

echo "Validate Fresh Repository"
./mobilecombackup validate --repo-root="$repodir"

echo "Import Test Data"
./mobilecombackup import --repo-root="$repodir" testdata

echo "Validate Repository after Successful Import"
./mobilecombackup validate --repo-root="$repodir"

echo "Should have imported calls"
if [[ $(ls -1 "$repodir/calls" | wc -l) -eq 0 ]]; then
	echo "ERROR: No calls found" >&2
	exit 2
fi

echo "Should have imported smses"
if [[ $(ls -1 "$repodir/sms" | wc -l) -eq 0 ]]; then
	echo "ERROR: No smses found" >&2
	exit 2
fi

echo "Should have imported attachments"
if [[ $(ls -1 "$repodir/attachments" | wc -l) -eq 0 ]]; then
	echo "ERROR: No attachments found" >&2
	exit 2
fi

echo "Re-Import Test Data should work"
./mobilecombackup import --repo-root="$repodir" testdata

echo "Validate Repository after Re-Import should work"
./mobilecombackup validate --repo-root="$repodir"

