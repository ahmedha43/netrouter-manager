NETROUTERD_VERSION = 0.1.0
NETROUTERD_SITE = $(BR2_EXTERNAL_NETROUTER_PATH)/../../native
NETROUTERD_SITE_METHOD = local
NETROUTERD_GOMOD = github.com/ahmedha43/netrouter-manager/native
NETROUTERD_LICENSE = Proprietary
NETROUTERD_BUILD_TARGETS = cmd/netrouterd cmd/netrouterctl
NETROUTERD_LDFLAGS = -s -w
# The native module also contains the optional Fyne Windows client. Buildroot's
# generated vendor metadata is intentionally bypassed for the service-only
# targets, which depend on the Go standard library and need no network fetches.
NETROUTERD_GO_ENV = GOFLAGS=-mod=mod

$(eval $(golang-package))
