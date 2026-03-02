.PHONY: run build clean docker

run: build
	@echo "Opening http://localhost:8000"
	@open http://localhost:8000 || true
	python3 -m http.server 8000

build: bmp.jsdos favicon.png

bmp.jsdos: build-bundle.sh dosbox.conf ../bmp/*
	./build-bundle.sh

favicon.png: ../images/favicon.png
	cp ../images/favicon.png .

docker: build
	podman build -t bmp-wasm .
	@echo "Run with: podman run -p 8090:80 bmp-wasm"

clean:
	rm -f bmp.jsdos version.txt favicon.png
