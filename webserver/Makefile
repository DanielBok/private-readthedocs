.phony: start stop build migrate

start: stop
	docker-compose -p psd up -d

stop:
	docker-compose -p psd down

build:
	go generate
	go build -o sphinx.exe .

migrate:
	migrate -path services/database/migrations -database postgres://postgres:password@localhost:5432/postgres?sslmode=disable up
