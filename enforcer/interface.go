package enforcer

// RouteModifier configures routing on a node to redirect traffic
type RouteModifier interface {

	// RouteToNode adds a route in a node to redirect traffic with destinationIP
	// to a targetNode
	// example destinationIP=192.168.56.22/32, targetNode=10.150.11.112
	RouteToNode(destinationIP, targetNode string) error

	// ClearRouteToNode removes the particular route rule
	ClearRouteToNode(destinationIP, targetNode string) error

	// RuleExists returns true if it exists, else an error
	RuleExists(destinationIP, targetNode string) (bool, error)
}

// SourceIPModifier configures the node which does SNAT for traffic
type SourceIPModifier interface {

	// AddNatRules add an iptables rule which SNATs all traffic for destinationIP
	// with NatVipIP. That is, any traffic with destination == destinationIP
	// leaving this node will have a source IP == NatVipIP
	AddNatRules(destinationIP, NatVipIP string) error

	// ClearNatRules clears IPtables rules added by AddNatRules
	ClearNatRules(detinationIP, NatVipIP string) error

	// ListNatRules lists Nat rules configured
	ListNatRules() ([]string, error)
}
