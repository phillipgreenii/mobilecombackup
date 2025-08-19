#!/bin/sh

set -e

echo ">> Build Project"
devbox run build-cli

echo ">> Create Repo Directory"
repodir=./tmp/full-test
rm -rf "$repodir"
mkdir -p $repodir

echo ">> Initialize Repository"
./mobilecombackup init --repo-root="$repodir"

echo ">> Validate Fresh Repository"
./mobilecombackup validate --repo-root="$repodir"

echo ">> Info should work with empty repository"
./mobilecombackup info --repo-root="$repodir"

echo ">> Setting adding contacts.yaml"
cat > "$repodir/contacts.yaml" << EOF
contacts: 
    - name: "Jim Henson"
      numbers:
        - "5555550004"
EOF
./mobilecombackup reprocess-contacts --repo-root="$repodir" 

echo ">> Validate After contact change"
./mobilecombackup validate --repo-root="$repodir" 

echo ">> Import Test Data"
./mobilecombackup import --repo-root="$repodir" testdata

echo ">> Validate Repository after Successful Import"
./mobilecombackup validate --repo-root="$repodir"

echo ">> Should have imported calls"
if [[ $(ls -1 "$repodir/calls" | wc -l) -eq 0 ]]; then
	echo "ERROR: No calls found" >&2
	exit 2
fi

echo ">> Should have calls with Jim Henson's phone number"
if [[ $( grep Henson -r "$repodir/calls" | wc -l) -eq 0 ]]; then
	echo "ERROR: Jim Henon's phone number is not in calls" >&2
	exit 2
fi

echo ">> Should have imported smses"
if [[ $(ls -1 "$repodir/sms" | wc -l) -eq 0 ]]; then
	echo "ERROR: No smses found" >&2
	exit 2
fi

echo ">> Should have imported attachments"
if [[ $(ls -1 "$repodir/attachments" | wc -l) -eq 0 ]]; then
	echo "ERROR: No attachments found" >&2
	exit 2
fi

echo ">> Should have unprocessed contacts"
if [[ $(yq --exit-status '.unprocessed[] | length > 0 ' "$repodir/contacts.yaml"  > /dev/null) -ne 0 ]]; then
	echo "ERROR: No unprocessed contacts found" >&2
	exit 2
fi


echo ">> Info should work with imported data"
./mobilecombackup info --repo-root="$repodir"

echo ">> Re-Import Test Data should work"
./mobilecombackup import --repo-root="$repodir" testdata

echo ">> Validate Repository after Re-Import should work"
./mobilecombackup validate --repo-root="$repodir"

