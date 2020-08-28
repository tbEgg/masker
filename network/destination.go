package network


// interface Destination implement interface net.Addr and Address
type Destination interface {
	Network()	string
	Address

	IsTCP() bool
	IsUDP() bool
}

func NewTCPDestination(address Address) Destination {
	return tcpDestination{
		Address:	address,
	}
}

func NewUDPDestination(address Address) Destination {
	return udpDestination{
		Address:	address,
	}
}


type tcpDestination struct {
	Address
}

func (tcpDestination) Network() string {
	return "tcp"
}

func (tcpDestination) IsTCP() bool {
	return true
}

func (tcpDestination) IsUDP() bool {
	return false
}


type udpDestination struct {
	Address
}

func (udpDestination) Network() string {
	return "udp"
}

func (udpDestination) IsTCP() bool {
	return false
}

func (udpDestination) IsUDP() bool {
	return true
}