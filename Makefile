.PHONY: build test dev docker-cover clean

APP := poterie
REGISTRY := ghcr.io/bdfrost
IMAGE := $(REGISTRY)/$(APP)

build:
	CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/$(APP) .

test:
	go test ./... -v -cover -count=1

dev:
	CGO_ENABLED=1 go run .

docker-build:
	docker build -t $(IMAGE):latest .

docker-push:
	docker push $(IMAGE):latest

clean:
	rm -rf bin/
	find . -name "*.db" -delete
