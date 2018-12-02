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

	// AddStaticIptablesRule adds rules for traffic from srcSet > dstSet to be SNAT
	// with NatVipIP.
	AddStaticIptablesRule(srcSetName, dstSetName, NatVipIP string) error

	// // ClearIptablesRule clears IPtables rules added by AddNatRules
	ClearIptablesRule(srcSetName, dstSetName, NatVipIP string) error

	// ListIptablesRule lists Nat rules configured
	ListIptablesRule() ([]string, error)

	// AddSourceIP
	AddSourceIP(ip string) error

	// DelSourceIP
	DelSourceIP(ip string) error

	// AddDestIP
	AddDestIP(ip string) error

	// DelDestIP
	DelDestIP(ip string) error
}
