AGENT=metric_agent
AGENT_PATH=./cmd/agent

SERVER=metric_collector
SERVER_PATH=./cmd/server

LINT=customlint
LINT_PATH=./cmd/staticlint

GO_BUILD=go build
DIST=dist
RMRF=rm -rf

VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')


LDFLAGS := -ldflags "\
    -X 'main.buildVersion=$(VERSION)' \
    -X 'main.buildCommit=$(COMMIT)' \
    -X 'main.buildDate=$(BUILD_DATE)'"

ALL_TARGETS := $(AGENT) $(SERVER) $(LINT)

.PHONY: all build clean $(ALL_TARGETS)

all: $(ALL_TARGETS)

$(DIST):
	mkdir -p $(DIST)

$(AGENT): $(DIST)
	@echo "    Building $(AGENT)..."
	@echo "    Version:      $(VERSION)"
	@echo "    Commit:       $(COMMIT)"
	@echo "    Build Date:   $(BUILD_DATE)"
	$(GO_BUILD) $(LDFLAGS) -o $(DIST)/$(AGENT) $(AGENT_PATH)

$(SERVER): $(DIST)
	@echo "    Building $(SERVER)..."
	@echo "    Version:      $(VERSION)"
	@echo "    Commit:       $(COMMIT)"
	@echo "    Build Date:   $(BUILD_DATE)"
	$(GO_BUILD) $(LDFLAGS) -o $(DIST)/$(SERVER) $(SERVER_PATH)

$(LINT): $(DIST)
	@echo "    Building $(LINT)..."
	@echo "    Version:      $(VERSION)"
	@echo "    Commit:       $(COMMIT)"
	@echo "    Build Date:   $(BUILD_DATE)"
	$(GO_BUILD) $(LDFLAGS) -o $(DIST)/$(LINT) $(LINT_PATH)

clean:
	@echo "    Cleaning..."
	$(RMRF) $(DIST)
