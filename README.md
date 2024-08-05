# NectGmbH - db-backup-controller

This controller is intended to act as a generic backup solution for all databases in the cluster.

It is deployed once and configured with one or more `DatabaseBackupStorageClass` defining where and how to store the backups. Each `DatabaseBackupStorageClass` can contain multiple locations (i.e. an S3 bucket and a GCS bucket for multiple storage locations).

When the `DatabaseBackupStorageClass`es are configured, `DatabaseBackup`s are created for each database to backup. These tell the controller which databases you want to have backed up into which `DatabaseBackupStorageClass`. Also these define how long to store the backups and how to retain backups (i.e. using a simple time-based retention or a grandfather-father-son retention).

After the controller is running and `DatabaseBackupStorageClass` as well as `DatabaseBackup` are configured the controller will create a new runner deployment inside your databases namespace which then will execute the backup in your configured interval.

As soon as that runner deployment is running you can `kubernetes exec` into it to trigger a backup immediately or trigger a restore of the backed up database to a point-in-time or to a specific backup.

## Deployment

The controller is built using Github Actions and published into the Github Container Registry as a Helm chart and as Docker images. The most simple way to deploy it is to just execute a Helm deployment:

```
helm upgrade \
  --create-namespace \
  --install \
  --namespace 'db-backup-controller' \
  --version '<release-tag, i.e. "0.5.0">' \
  db-backup-controller \
  oci://ghcr.io/nectgmbh/db-backup-controller
```

The chart as well as the images are published using the `git describe --tags --always` version format while tags are created in SemVer. So there are `v1.2.3` tags resulting in `...:1.2.3` tags for the Helm chart and the Docker images. You would deploy that using `--version 1.2.3`. Additionally there are `...:1.2.3-g56789ab` tags for development builds. Please advice: Those are not guaranteed to be stable, so do not use them for important databases.

## Development

### Code Generation

The controller relies on generated CRDs and Kubernetes code. After changing anything within the `pkg/apis/...` directory you need to re-generate the CRDs and the Kubernetes code using `make generate-code`.

For this to work you need a fully set up Go development environment. Have a look at the [`Makefile`](./Makefile) for details around the code-generation.

### Adding a new Backup-Engine

- Have a look at the [`pkg/backupengine/interface.go`](./pkg/backupengine/interface.go) definition
- See for example the `cockroach` (advanced) or `postgres` (simple) engines for implementation examples
- Create a new `pkg/backupengine/<yourengine>/...` package and implement the interface
- Create a `Dockerfile` or some `Dockerfile.<version>` inside your package for the `backup-runner` image
- Register your new engine in the [`pkg/backupengine/registry.go`](./pkg/backupengine/registry.go)
- Edit the [`DatabaseBackupSpec`](./pkg/apis/v1/types.go) to add the config for your new engine
- Build all the docker-stuff (including your new engine runner-image)

### Local Test Deployment

For local testing a Minikube cluster is recommended. Also you need a registry to push the images to which is accessible from your local machine and the cluster you've set up. As your registry probably differs from the value of the `LOCAL_IMAGE` variable in the `Makefile` you need to export your local-image-prefix as `LOCAL_IMAGE` (i.e. `export LOCAL_IMAGE=http://10.0.0.1:5000/db-backup-controller:latest`) before executing the following make-targets:

- `make deploy-local` - Builds all Docker images required, pushes them into the registry mentioned in the `LOCAL_IMAGE` and then deploys the `db-backup-controller` chart into your current Kubernetes context (which should be the Minikube)
- `make deploy-testenv` - Deploys a test environment consisting of a CockroachDB including test data, MinIO, `DatabaseBackupStorageClass` and `DatabaseBackup` into your current Kubernetes context. When setting `TESTENV_HUGEDATA=true` during the deployment a large amount of data is inserted into the database which might take a while. This is intended to test big backups so make sure you do have enough storage space to accomodate for all the data.
- `make force-local-backup` - Executes a backup in the testenv (will fail when testenv is not deployed). See the logs of the backup-runner in the `db-backup-controller-testenv` namespace for its current status.
- `make force-local-dbdelete` - Removes the `database` database from the CRDB instance so the `force-local-restore` can indeed restore the database
- `make force-local-restore` - Executes a restore of the last backup of the testenv. Again look for logs in the runner in the namespace. It will fail if you don't delete the database before executing the restore (intended behavior as CRDB refuses to restore over an existing database).


