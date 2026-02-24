.PHONY: build run stop logs

MAKE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

build:
	docker-compose -f $(MAKE_DIR)/docker-compose.yml build

run:
	docker-compose -f $(MAKE_DIR)/docker-compose.yml up -d

stop:
	docker-compose -f $(MAKE_DIR)/docker-compose.yml down

logs:
	docker-compose -f $(MAKE_DIR)/docker-compose.yml logs -f
