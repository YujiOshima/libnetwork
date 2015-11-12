package ipvlan

var (
	// driver mode UX option
	driverModeOpt = networkType + modeOpt
)

const (
	// macvlan mode private
	modeL3 = "l3"
	// macvlan mode vepa
	modeL2 = "l2"
	// host interface UX key
	hostIfaceOpt = "host_iface"
	// macvlan mode UX opt suffix
	modeOpt = "_mode"
	// The default macvlan mode
	defaultMacvlanMode = "l2"
)
