# Project automation commands

PYTHON ?= python3
PIP ?= $(PYTHON) -m pip
RUFF ?= ruff
PYTEST ?= pytest
SRC_DIR ?= src

.PHONY: help install lint format test check run clean

help:
	@echo "Available targets:"
	@echo "  make install   Install package requirements and tooling"
	@echo "  make lint      Run Ruff checks"
	@echo "  make format    Format code with Ruff"
	@echo "  make test      Run pytest"
	@echo "  make check     Run linting and tests"
	@echo "  make run       Start the Syncthing kicker service"
	@echo "  make clean     Remove cache artifacts"

install:
	$(PIP) install --upgrade pip
	$(PIP) install -r requirements.txt
	$(PIP) install pytest ruff

lint:
	$(RUFF) check $(SRC_DIR)

format:
	$(RUFF) format $(SRC_DIR)

# pytest exits with code 5 when it finds no tests; treat that as success for now.
test:
	$(PYTEST) || [ $$? -eq 5 ]

check: lint test

run:
	$(PYTHON) $(SRC_DIR)/main.py

clean:
	rm -rf __pycache__ .pytest_cache .ruff_cache
