GO := /usr/local/go/bin/go
INSTALL := /usr/bin/install

BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin
TEST_DIR := $(BUILD_DIR)/test

# Set to -v for verbose o/p. May set it in Env before calling Make
# export TEST_FLAG=-v in shell before calling make or
# TEST_FLAG=-v make all

export GO111MODULE=on

SRC_FILES=$(shell find . -name '*.go' | grep -v '_test.go' | grep -v '/tests/')
TEST_FILES=$(wildcard *_test.go)

# Binaries used in LoM system
ENGINE_BIN := $(BIN_DIR)/LoMEngine


# Test binaries/flags
LIB_TEST = $(TEST_DIR)/LibTested
ENGINE_TEST = $(TEST_DIR)/EngineTested
SONIC_DBCLIENT_TEST =$(TEST_DIR)/SonicDbClientTested
SONIC_LINK_CRC_TEST = $(TEST_DIR)/SonicLinkCrcTested
all: $(ENGINE_BIN) $(LIB_TEST) $(ENGINE_TEST) $(SONIC_DBCLIENT_TEST) $(SONIC_LINK_CRC_TEST)


$(ENGINE_BIN): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(BIN_DIR)
	$(GO) build  -o $@ lom/src/engine
	$(info "Built lom-engine")


$(LIB_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/LibCoverprofile.out \
		-coverpkg lom/src/lib/lomipc,lom/src/lib/lomcommon \
		-covermode=atomic ./src/lib/lib_test || (echo "LibTest failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/LibCoverprofile.out -o $(TEST_DIR)/LibCoverprofile.html
	touch $@


$(ENGINE_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/EngineCoverprofile.out \
		-coverpkg lom/src/engine \
		-covermode=atomic lom/src/engine || (echo "EngineTest failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/EngineCoverprofile.out -o $(TEST_DIR)/EngineCoverprofile.html
	touch $@

$(SONIC_DBCLIENT_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/SonicDbClientCoverprofile.out \
		-coverpkg lom/src/plugins/sonic/client/dbclient \
		-covermode=atomic lom/src/plugins/sonic/client/dbclient || (echo "SonicDbClient Test failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/SonicDbClientCoverprofile.out -o $(TEST_DIR)/SonicDbClientCoverprofile.html
	touch $@

$(SONIC_LINK_CRC_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/SonicLinkCrcCoverprofile.out \
                -coverpkg lom/src/plugins/sonic/plugin/detection/link/linkcrc \
                -covermode=atomic lom/src/plugins/sonic/plugin/detection/link/linkcrc || (echo "SonicLinkCrc plugin Test failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/SonicLinkCrcCoverprofile.out -o $(TEST_DIR)/SonicLinkCrcCoverprofile.html
	touch $@

clean:
	rm -rf $(BUILD_DIR)
