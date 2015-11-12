package macvlan

var (
	// driver mode UX option
	driverModeOpt = networkType + modeOpt
)

const (
	// macvlan mode private
	modePrivate = "private"
	// macvlan mode vepa
	modeVepa = "vepa"
	// macvlan mode bridge
	modeBridge = "bridge"
	// macvlan mode passthrough
	modePassthru = "passthru"
	// host interface UX key
	hostIfaceOpt = "host_iface"
	// macvlan mode UX opt suffix
	modeOpt = "_mode"
	// The default macvlan mode
	defaultMacvlanMode = "bridge"
)
