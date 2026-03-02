.PHONY: build run stop

IMAGE_NAME := bmp
CONTAINER_NAME := bmp
PORT := 8080

build:
	podman build -t $(IMAGE_NAME) .

run: build
	./run_local.sh

stop:
	podman stop $(CONTAINER_NAME) 2>/dev/null || true
	podman rm $(CONTAINER_NAME) 2>/dev/null || true
