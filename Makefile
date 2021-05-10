PRODUCT_MAJOR_VERSION=6
PRODUCT_MINOR_VERSION=6
PRODUCT_MICRO_VERSION=80
PRODUCT_VERSION = $(PRODUCT_MAJOR_VERSION).$(PRODUCT_MINOR_VERSION).$(PRODUCT_MICRO_VERSION)

GO_BUILD = go build
GO_FLAGS = -v

REPO = $(shell pwd)
SOURCEDIR = $(REPO)/mk
BUILDDIR = $(REPO)/build
GOBUILDDIR = $(BUILDDIR)/gobuild
STAGEDIR = $(BUILDDIR)/stage
OBJECTDIR = $(BUILDDIR)/obj
DISTDIR = $(BUILDDIR)/dist

OBJECTS :=
OBJECTS += $(OBJECTDIR)/xe-daemon
OBJECTS += $(OBJECTDIR)/xenstore

PACKAGE = xe-guest-utilities
VERSION = $(PRODUCT_VERSION)
RELEASE := $(shell git rev-list HEAD | wc -l)
ifeq ($(GOARCH),)
        ARCH := $(shell go version|awk -F'/' '{print $$2}')
else
        ARCH := $(GOARCH)
endif

ifeq ($(ARCH), amd64)
	ARCH = x86_64
endif

XE_DAEMON_SOURCES :=
XE_DAEMON_SOURCES += xe-daemon/xe-daemon.go
XE_DAEMON_SOURCES += syslog/syslog.go
XE_DAEMON_SOURCES += system/system.go
XE_DAEMON_SOURCES += guestmetric/guestmetric.go
XE_DAEMON_SOURCES += guestmetric/guestmetric_linux.go
XE_DAEMON_SOURCES += xenstoreclient/xenstore.go

XENSTORE_SOURCES :=
XENSTORE_SOURCES += xenstore/xenstore.go
XENSTORE_SOURCES += xenstoreclient/xenstore.go

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
	  ln -sf xenstore $(STAGEDIR)/usr/bin/xenstore-read ; \
	  ln -sf xenstore $(STAGEDIR)/usr/bin/xenstore-write ; \
	  ln -sf xenstore $(STAGEDIR)/usr/bin/xenstore-exists ; \
	  ln -sf xenstore $(STAGEDIR)/usr/bin/xenstore-rm ; \
	  ln -sf xenstore $(STAGEDIR)/usr/bin/xenstore-list ; \
	  ln -sf xenstore $(STAGEDIR)/usr/bin/xenstore-ls ; \
	  ln -sf xenstore $(STAGEDIR)/usr/bin/xenstore-chmod ; \
	  ln -sf xenstore $(STAGEDIR)/usr/bin/xenstore-watch ; \
	  install -d $(STAGEDIR)/etc/udev/rules.d/ ; \
	  install -m 644 $(SOURCEDIR)/xen-vcpu-hotplug.rules $(STAGEDIR)/etc/udev/rules.d/z10_xen-vcpu-hotplug.rules ; \
	  cd $(STAGEDIR) ; \
	  tar zcf $@ * \
	)

$(OBJECTDIR)/xe-daemon: $(XE_DAEMON_SOURCES:%=$(GOBUILDDIR)/%)
	mkdir -p $(OBJECTDIR)
	$(GO_BUILD) $(GO_FLAGS) -o $@ $<

$(OBJECTDIR)/xenstore: $(XENSTORE_SOURCES:%=$(GOBUILDDIR)/%) $(GOROOT)
	mkdir -p $(OBJECTDIR)
	$(GO_BUILD) $(GO_FLAGS) -o $@ $<

$(GOBUILDDIR)/%: $(REPO)/%
	mkdir -p $$(dirname $@)
	cat $< | \
	sed -e "s/@PRODUCT_MAJOR_VERSION@/$(PRODUCT_MAJOR_VERSION)/g" | \
	sed -e "s/@PRODUCT_MINOR_VERSION@/$(PRODUCT_MINOR_VERSION)/g" | \
	sed -e "s/@PRODUCT_MICRO_VERSION@/$(PRODUCT_MICRO_VERSION)/g" | \
	sed -e "s/@NUMERIC_BUILD_NUMBER@/$(RELEASE)/g" \
	> $@

