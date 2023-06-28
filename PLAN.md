# The PLAN

- Have a **controller** (=> `./cmd/backup-controller`)
- Have a **runner** (=> `./cmd/backup-runner`)
- Have custom resources (=> `./pkg/apis/...`)
    - Backup storage location + secrets - MUST exist globally / in controller namespace
    - Backup definition + secrets - MUST exist in db/app namespace

## Controller

Controller is a quite dumb thingy, running as a K8s deployment, watching custom resources…

- Controller gets a change of a backup definition
- Controller asks the requested engine (MySQL, Postgres, …) to yield a pod definition (=> `./pkg/backupengine/...`)
    - Engine must make sure all required secrets are mounted in using its pod definition
- Serializes the storage location and backup definition CRs and adds them as secret-mount to the pod
    - STS must restart when config secret is updated
    - Secret is written & updated to controller Namespace
    - Config secret contains the referenced contents of other secrets
- Controller maintains a STS containing that runner-pod definition, a secret for the config, and a service **in the controller namespace**
    - imagePullSecrets are added by controller to the spec as everything is running in controller ns
    - optionally a Pod-Affinity is added to have the runner run on the same machine as the DB (preferred during schedule)
    - Secret name: runner-{{ namespace }}-{{ name }}
    - STS name: runner-{{ namespace }}-{{ name }}
    - SVC name: runner-{{ namespace }}-{{ name }} (used i.e. for cockroach to send backup via http)
- Stuff changes? Controller changes the STS.
- Stuff vanishes? Controller gets rid of the STS.

## Runner

Runner is a program placed inside a container surrounded by the real backup tools (mysqldump, pg\_dump, …) and acts as scheduler / manager / uploader.

- Has a config provided by the controller
    - equals CR but with filled in secrets so runner does not need to talk to cluster
- Reads config
- Waits for time to come
- Asks requested engine to take a backup (=> `./pkg/backupengine/...`)
    - Engine knows what to execute to backup `$db` from `$host` with `$credentials`
    - Engine stores backup to file location it is given by runner
- Uploads backup to storage location (=> `./pkg/storage/...`)
    - Upload location in the bucket is a generated name from the backup definition name and the namespace
- Takes notes which backups exist, manages "labels" for them, if no more labels are attached removes backup
    - Can run in "single backup" mode: No labels, no management, no retention, just a single uploaded target
- Goto "wait" and repeat
- Can be asked to restore a backup
    - Downloads backup (=> `./pkg/storage/...`)
    - Askes engine to restore that backup (=> `./pkg/backupengine/...`)
    - Engine knows what to execute to restore `$db` to `$host` with `$credentials` from file

## Progress

- CRDs => Present, probably not final
- Controller => Present, runnable, creates resources
- Runner => Present, untested
- Backup Engines
    - MySQL => 404
    - Postgres => 404
    - Cockroach => Present, untested
- Storage
    - S3 => Present, untested
- Label Manager => Present, probably needs refinement, should work

## FAQ

- **Why STS, not a Deployment?**  
  Simple: Deployment even though using RWOnce volume and delete-first-start-after strategy does not guarantee the old pod is dead before starting the new one. STS guarantees it's gone before starting a new one. As those two share a storage they MUST NOT run concurrently. That might mess up the label storage, that might mess up the backup, that might run concurrent backups which then are partially broken. They just MUST NOT run at the same time. So leave it to Kubernetes to ensure the old king died before proclaiming the new king…
