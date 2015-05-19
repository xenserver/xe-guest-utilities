
GO_BUILD = go build
GO_FLAGS = -a -x

SRC_DIR = .
BIN_DIR = bin

BINARIES :=
BINARIES += $(BIN_DIR)/xe-daemon
BINARIES += $(BIN_DIR)/xenstore

XE_DAEMON_SOURCES :=
XE_DAEMON_SOURCES += $(SRC_DIR)/xe-daemon/xe-daemon.go
XE_DAEMON_SOURCES += $(SRC_DIR)/guestmetric/guestmetric.go
XE_DAEMON_SOURCES += $(SRC_DIR)/guestmetric/guestmetric_linux.go
XE_DAEMON_SOURCES += $(SRC_DIR)/xenstoreclient/xenstore.go

XENSTORE_SOURCES :=
XENSTORE_SOURCES += $(SRC_DIR)/xenstore/xenstore.go
XENSTORE_SOURCES += $(SRC_DIR)/xenstoreclient/xenstore.go

.PHONY: build
build: $(BINARIES)

.PHONY: clean
clean:
	-rm -f $(BINARIES)

$(BIN_DIR)/xe-daemon: $(XE_DAEMON_SOURCES)
	mkdir -p $(BIN_DIR)
	$(GO_BUILD) $(GO_FLAGS) -o $@ $<

$(BIN_DIR)/xenstore: $(XENSTORE_SOURCES)
	mkdir -p $(BIN_DIR)
	$(GO_BUILD) $(GO_FLAGS) -o $@ $<

