
PRODUCT_VERSION = 6.6.80

GO_BUILD = go build
GO_FLAGS = -a -x

REPO = $(shell pwd)
SOURCEDIR = $(REPO)/mk
BUILDDIR = $(REPO)/build
STAGEDIR = $(BUILDDIR)/stage
OBJECTDIR = $(BUILDDIR)/obj
DISTDIR = $(BUILDDIR)/dist

OBJECTS :=
OBJECTS += $(OBJECTDIR)/xe-daemon
OBJECTS += $(OBJECTDIR)/xenstore

PACKAGE = xe-guest-utilities
VERSION = $(PRODUCT_VERSION)
RELEASE := $(shell git rev-list HEAD | wc -l)
ARCH := $(shell go version|awk -F'/' '{print $$2}')

ifeq ($(ARCH), amd64)
	ARCH = x86_64
endif

XE_DAEMON_SOURCES :=
XE_DAEMON_SOURCES += $(REPO)/xe-daemon/xe-daemon.go
XE_DAEMON_SOURCES += $(REPO)/guestmetric/guestmetric.go
XE_DAEMON_SOURCES += $(REPO)/guestmetric/guestmetric_linux.go
XE_DAEMON_SOURCES += $(REPO)/xenstoreclient/xenstore.go

XENSTORE_SOURCES :=
XENSTORE_SOURCES += $(REPO)/xenstore/xenstore.go
XENSTORE_SOURCES += $(REPO)/xenstoreclient/xenstore.go

.PHONY: build
build: $(DISTDIR)/$(PACKAGE)_$(VERSION)-$(RELEASE)_$(ARCH).tgz

.PHONY: clean
clean:
	$(RM) -rf $(BUILDDIR)

$(DISTDIR)/$(PACKAGE)_$(VERSION)-$(RELEASE)_$(ARCH).tgz: $(OBJECTS)
	( mkdir -p $(DISTDIR) ; \
	  install -d $(STAGEDIR)/etc/init.d/ ; \
	  install -m 755 $(SOURCEDIR)/xe-linux-distribution.init $(STAGEDIR)/etc/init.d/xe-linux-distribution ; \
	  install -d $(STAGEDIR)/usr/sbin/ ; \
	  install -m 755 $(SOURCEDIR)/xe-linux-distribution $(STAGEDIR)/usr/sbin/xe-linux-distribution ; \
	  install -m 755 $(OBJECTDIR)/xe-daemon $(STAGEDIR)/usr/sbin/xe-daemon ; \
	  install -d $(STAGEDIR)/usr/bin/ ; \
	  install -m 755 $(OBJECTDIR)/xenstore $(STAGEDIR)/usr/bin/xenstore ; \
	  ln -s /usr/bin/xenstore $(STAGEDIR)/usr/bin/xenstore-read ; \
	  ln -s /usr/bin/xenstore $(STAGEDIR)/usr/bin/xenstore-write ; \
	  ln -s /usr/bin/xenstore $(STAGEDIR)/usr/bin/xenstore-exists ; \
	  ln -s /usr/bin/xenstore $(STAGEDIR)/usr/bin/xenstore-rm ; \
	  install -d $(STAGEDIR)/etc/udev/rules.d/ ; \
	  install -m 644 $(SOURCEDIR)/xen-vcpu-hotplug.rules $(STAGEDIR)/etc/udev/rules.d/z10_xen-vcpu-hotplug.rules ; \
	  cd $(STAGEDIR) ; \
	  tar cf $@ * \
	)

$(OBJECTDIR)/xe-daemon: $(XE_DAEMON_SOURCES)
	mkdir -p $(OBJECTDIR)
	$(GO_BUILD) $(GO_FLAGS) -o $@ $<

$(OBJECTDIR)/xenstore: $(XENSTORE_SOURCES)
	mkdir -p $(OBJECTDIR)
	$(GO_BUILD) $(GO_FLAGS) -o $@ $<

