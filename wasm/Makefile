.PHONY: run build clean docker run-docker push

build: bmp.jsdos favicon.png

bmp.jsdos: build-bundle.sh dosbox.conf ../bmp/*
	./build-bundle.sh

favicon.png: ../images/favicon.png
	cp ../images/favicon.png .

docker: build
	podman build -t bmp-wasm .

run-docker: docker
	@mkdir -p /tmp/bmp-saves
	@echo "Opening http://localhost:8090"
	@open http://localhost:8090 || true
	podman run --rm -p 8090:8080 -v /tmp/bmp-saves:/data bmp-wasm

push: build
	./build-and-push-docker-image.sh

run: build
	@echo "Opening http://localhost:8000 (no save persistence)"
	@open http://localhost:8000 || true
	python3 -m http.server 8000

clean:
	rm -f bmp.jsdos version.txt favicon.png
