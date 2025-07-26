# Specification

## Overview

Each day, SMS and Call logs are backed up on phone.  Historically, the backup would include all SMS and calls whaich are on the phone. Later backups include entries from the previous X number days.  Either way backups contain a lot of duplicates.  This is a major problem with SMS which backup the same images over and over again.  This leads to daily backup files of >700MB.  

To remedy this, this code will focus on doing two things.
 1) Remove duplicates.  We should be able to take all of the entries and keep only distinct ones. The format of the entries may make this difficult due to the same entry being stored in the backup in different ways.
    - Fields may change over time
    - Timestamp field with timezone uses the current location, which might be different from a previous backup.
 2) Store images external of the backup files.
    - By default, the images are stored in the backup files encoded with base64.
    - We want to change this so that Images will be written to disk in a location identified by hash.

### Expected Flow

1. Load Files
   - Read Exisiting Repository
   - Any specified unprocessed backup file
2. Entry Transformation
   - Extract Images
   - Remove Duplicates
3. Order Entries
   - Sort by Timestamp (UTC)
   - Partition by Year
4. Overwrite Repository

## File / Directory Schemas

### Call Backup (`calls.xml`)

Calls are stored as XML, schema is as follows:

- `calls`
  - attributes:
    - `count`: number of call entries in this file
  - children:
    - `call`
      - attributes:
        - `number`: phone number
          - possible formats: `ddddddd`, `dddddddddd`, `+1dddddddddd`
        - `duration`: duration of the call in seconds
        - `date`: timestamp in epoch seconds
        - `type`: enum to represent the type of call
          - TODO: define meaning of each type
          - 1: ?
          - 2: ?
          - 3: ?
          - 4: ?
        - `readlable_date`: `date` formated to be consumed by humans; may not be consistent, don't use in comparison
        - `contact_name`: name associated with phone number; may not be consistent, don't use in comparisonA

### SMS Backup (`sms.xml`)

__TODO: define SMS backup schema__

### Repository

A repository is a file structure which will house all of the processed backup files.  The structure is as follows:

```
repository/
├── contacts.yaml
├── files.yaml
├── files.yaml.sha256
├── summary.yaml
├── calls/
│   ├── calls-2015.xml
│   └── calls-2016.xml
├── sms/
│   ├── sms-2015.xml
│   └── sms-2016.xml
└── attachments/
    └── [hash-based subdirectories]
```

#### contacts.yaml

`contacts.yaml` contains mapping from number to contact name.

```yaml
contacts:
  - name: "Bob Ross"
    numbers: 
      - "+11111111111"
  - name: "<unknown>"
    numbers:
      - "8888888888"
      - "9999999999"
```

#### files.yaml

`files.yaml` lists all files in the repository with their associated sha256.
 - Exclusions: `files.yaml` and `files.yaml.sha256`

```yaml
files:
  - file: summary.yaml
    sha256: ...
  - file: contacts.yaml
    sha256: ...
  - file: calls/calls-2015.xml
    sha256: ...
```

#### files.yaml.sha256

`files.yaml.sha256` is the SHA256 of `files.yaml`.

#### summary.yaml

`summary.yaml` contains a summary of repository contents.

|Property|Description|
|--------|-----------|
| `count`  | Total number of record of thie type |
| `size_bytes` | Total size, in bytes, of the records for this type. |
| `oldest`  | Timestamp of the oldest record of this type |
| `latest`  | Timestamp of the latest record of this type |

```yaml
summary:
  contacts:
    count: 12
    size_bytes: 1000
  calls:
    count: 234
    oldest: $timestamp
    latest: $timestamp2
    size_bytes: 2000
  sms:
    count: 2345
    oldest: $timestamp3
    latest: $timestamp4
    size_bytes: 123456
  attachments:
    count: 12
    size_bytes: 1000000
```

#### calls/

These files are of the same schema as outlined previousl in "Calls Backup".

There should be a file per year: 
 * `calls-2015.xml`
 * `calls-2016.xml`

#### sms/

These files are of the same schema as outlined previousl in "SMS Backup".

There should be a file per year: 

 * `sms-2015.xml`
 * `sms-2016.xml`

#### attachments/

Contents will be the attachments identified by their hash.  They will be grouped into directories based upon the first 2 characters of the hash:

 * `ab`
   * `ab54363e39`
   * `ab8bd623a0`
 * `b2`
   * `b28fe7c6ab`

