ENGINE_IMAGES:=$(shell find ./pkg/backupengine -name 'Dockerfile*' | sed -E 's@.*/([^/]+)/Dockerfile.*@\1@' | sort | uniq)
CODE_GEN_VERSION:=v0.28.1
HELM:=helm3
LOCAL_IMAGE:=registry.local.nect/db-backup-controller:t$(shell date +%s)
TEST_RUNNER_NAME:=sha1-05eb843e76925a21a52bdc1acad03288810c87bf-0
TESTENV_HUGEDATA:=false

export CONTROLLER_GEN_VERSION:=v0.11.3

default: generate-code

# generate-code updates the CRDs in the Helm chart and generates the
# contents of the ./pkg/generated folder. Contents of the respective
# directories is wiped in before!
generate-code: ci/controller-gen ci/code-generator
	rm -rf \
		./charts/db-backup-controller/crds/*.yaml \
		./pkg/generated
	ci/controller-gen \
		crd output:crd:dir=./charts/db-backup-controller/crds \
		paths=./pkg/apis/...
	bash ci/codegen.sh

# build-engine-images is a shortcut for the CI to trigger a build and
# push for all images required by the engines
build-engine-images: $(patsubst %,build-engine-image-%,$(ENGINE_IMAGES))
build-engine-image-%:
	[ -n "$(DRONE_COMMIT_SHA)" ] || [ -n "$(IMAGE_NAME_OVERRIDE)" ] # Can only build when DRONE_COMMIT_SHA is present
	bash ./ci/push-engine-image.sh $*

# --- Local setup

docker-build-local:
	docker build -t $(LOCAL_IMAGE) .
	docker push $(LOCAL_IMAGE)
	IMAGE_NAME_OVERRIDE=$(LOCAL_IMAGE) $(MAKE) build-engine-images

deploy-local: docker-build-local
	$(HELM) upgrade \
		--create-namespace \
		--install \
		--namespace db-backup-controller \
		--set image=$(LOCAL_IMAGE) \
		--set imagePullPolicy=Always \
		--set jsonLog=false \
		--set logLevel=debug \
		--set rescanInterval=1m \
		--wait \
		db-backup-controller \
		./charts/db-backup-controller

deploy-testenv:
	$(HELM) upgrade \
		--create-namespace \
		--install \
		--namespace db-backup-controller-testenv \
		--set hugeDataGenerator.enabled=$(TESTENV_HUGEDATA) \
		--wait \
		db-backup-controller-testenv \
		./charts/testenv

force-local-backup:
	kubectl -n db-backup-controller exec -ti $(TEST_RUNNER_NAME) -- \
		/usr/local/bin/backup-runner backup

force-local-restore:
	kubectl -n db-backup-controller exec -ti $(TEST_RUNNER_NAME) -- \
		/usr/local/bin/backup-runner restore $(shell date --iso-8601=seconds)

# --- Build tooling

ci/controller-gen:
	bash ci/build-controller-gen "$(CURDIR)/ci/controller-gen"

ci/code-generator:
	git clone \
		--depth=1 \
		-b $(CODE_GEN_VERSION) \
		https://github.com/kubernetes/code-generator.git $@
