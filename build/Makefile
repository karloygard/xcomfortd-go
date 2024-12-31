# This makefile expects that the docker buildx addon is installed: https://github.com/docker/buildx

REPO:="karloygard"
IMAGE:="xcomfortd"
TAG:="0.66"

BUILD_ARGS:="--push"
BUILDER_NAME:="mybuilder"
VPATH:="../"

.DEFAULT_GOAL:= help
.PHONY:="docker-builder-prepare docker-builder-create docker-build help"

docker-builder-prepare: ## Prepare multi arch builds. This needs to be done every time you reboot your machine!
	@docker run --rm --privileged docker/binfmt:a7996909642ee92942dcd6cff44b9b95f08dad64

docker-builder-create: ## Create a new builder. This only needs to be done once!
	@docker buildx create --name ${BUILDER_NAME}
	@docker buildx use ${BUILDER_NAME}
	@docker buildx inspect --bootstrap

docker-build: ## Build multi arch image and push to docker hub.
	@cd .. && \
	docker buildx build \
		${BUILD_ARGS} \
		--platform linux/amd64,linux/arm64,linux/ppc64le,linux/386,linux/arm/v7,linux/arm/v6 \
		-t ${REPO}/${IMAGE}:latest \
		-f build/Dockerfile \
		.
	
	@cd .. && \
	docker buildx build \
		${BUILD_ARGS} \
		--platform linux/amd64,linux/arm64,linux/ppc64le,linux/386,linux/arm/v7,linux/arm/v6 \
		-t ${REPO}/${IMAGE}:${TAG} \
		-f build/Dockerfile \
		.

help: ## Show target descriptions.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
