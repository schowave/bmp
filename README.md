# README
Bundesliga Manager Professional as js-dos

# Create Executable
- Go to [DOS-Zone](https://dos.zone/studio/) and create jsdos file.
-- Place this file under jsdos-bmp/bundle.jsdos

# Docker Container

````bash
docker build -t schowave:bmp .
````

````bash
docker run --rm -p 127.0.0.1:8080:3000 --env PORT=3000 --name bmp schowave:bmp 
````

# Export Docker Container
````bash
docker save -o bmp.tar schowave:bmp
````
