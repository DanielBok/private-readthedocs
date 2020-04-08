.phony: start stop build

start: stop
	docker-compose -p psd up -d

stop:
	docker-compose -p psd down

build:
	go build -o sphinx.exe .
