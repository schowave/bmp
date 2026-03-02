.PHONY: run build clean

run: build
	@echo "Opening http://localhost:8000"
	@open http://localhost:8000 || true
	python3 -m http.server 8000

build: bmp.jsdos

bmp.jsdos: build-bundle.sh dosbox.conf ../bmp/*
	./build-bundle.sh

clean:
	rm -f bmp.jsdos version.txt
