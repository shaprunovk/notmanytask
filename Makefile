all: web crashme setup

protos:
	true

make_build:
	mkdir -p build

web: make_build
	go build -o build ./cmd/web

crashme: make_build
	CGO_ENABLED=0 go build -ldflags="-extldflags=-static" -o build ./cmd/crashme

run_web: web
	./build/web

coverage_html:
	go test ./... -coverprofile coverage.out
	go tool cover -html=coverage.out -o coverage.html
	rm coverage.out

docker_image:
	docker build . -f Dockerfile -t bigredeye/notmanytask:latest --platform=linux/amd64

docker_hub: docker_image
	docker push bigredeye/notmanytask:latest

docker_image_crashme:
	docker build . -f Dockerfile.crashme -t bigredeye/notmanytask:crashme --platform=linux/amd64

docker_hub_crashme: docker_image_crashme
	docker push bigredeye/notmanytask:crashme



ANSIBLE_PLAYBOOK=register_runners.yml
DOCKER_COMPOSE_FILE=docker-compose.yml
REGISTRATION_TOKEN ?= your_runners_token_here

setup: pull-images start register-runners

pull-images:
	docker-compose -f $(DOCKER_COMPOSE_FILE) pull

start:
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d platform


register-runners:
	ansible-playbook -e "registration_token=$(REGISTRATION_TOKEN)" $(ANSIBLE_PLAYBOOK)

clean:
	docker-compose -f $(DOCKER_COMPOSE_FILE) down


restart: clean setup


logs:
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

help:
	@echo "Usage: make [target]"
	@echo "Targets:"
	@echo "  all               Setup runners and build"
	@echo "  pull-images       Pull Docker images"
	@echo "  start       Start service"
	@echo "  register-runners  Register runners using Ansible"
	@echo "  clean             Stop and clean up Docker containers"
	@echo "  restart           Restart service"
	@echo "  logs              Show logs for Docker containers"
	@echo "  help              Show this help message"

