.PHONY: build run stop push

IMAGE_NAME := bmp
CONTAINER_NAME := bmp
PORT := 8080

build:
	podman build -t $(IMAGE_NAME) .

run: stop build
	podman run -d \
		--name $(CONTAINER_NAME) \
		-p $(PORT):8080 \
		-v $(PWD)/savegame:/savegame \
		$(IMAGE_NAME)
	@echo "Running — open http://localhost:$(PORT)"

stop:
	podman stop $(CONTAINER_NAME) 2>/dev/null || true
	podman rm $(CONTAINER_NAME) 2>/dev/null || true

push:
	./build-and-push-docker-image.sh
