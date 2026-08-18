# mobilecombackup

> Process mobile phone backup files: deduplicate call logs and SMS/MMS messages and organize them by year.
> More information: <https://github.com/phillipgreenii/mobilecombackup>.

- Initialize a new repository in the current directory:

`mobilecombackup init`

- Initialize a repository in a specific directory:

`mobilecombackup init --repo-root {{path/to/repo}}`

- Preview what `init` would create, without writing anything:

`mobilecombackup init --dry-run`

- Import backup files (`calls*.xml`, `sms*.xml`) into a repository:

`mobilecombackup import --repo-root {{path/to/repo}} {{path/to/backup1.xml}} {{path/to/backup2.xml}}`

- Import every backup file found under a directory:

`mobilecombackup import --repo-root {{path/to/repo}} {{path/to/backup_directory}}`

- Validate a repository's structure, manifests, checksums, and cross-references:

`mobilecombackup validate --repo-root {{path/to/repo}}`

- Validate and automatically fix the violations that are safe to fix:

`mobilecombackup validate --repo-root {{path/to/repo}} --autofix`

- Show repository statistics as JSON:

`mobilecombackup info --repo-root {{path/to/repo}} --json`

- Re-extract contacts from the existing backup data after editing `contacts.yaml`:

`mobilecombackup reprocess-contacts --repo-root {{path/to/repo}}`

- Generate a shell completion script:

`mobilecombackup completion {{bash|zsh|fish|powershell}}`
