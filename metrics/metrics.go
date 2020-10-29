package metrics

type MetricsManager interface {
	ReceiveResults(ResMsg) error
}

// ResMsg holds all the data received from a scan.
type ResMsg struct {
	Name            string
	IP              string
	Protocol        string
	OpenPorts       []string
	UnexpectedPorts []string
	ClosedPorts     []string
}
