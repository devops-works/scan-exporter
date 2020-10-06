package metrics

type resMsg struct {
	id              string
	ip              string
	protocol        string
	openPorts       []string
	unexpectedPorts []string
	closedPorts     []string
}

// Handle receives data from a finished scan. It also receive the number of targets declared in config file
func Handle(res resMsg, nTargets int) {
	// check if there is already some entries in redis
	// write data in target:ip:proto:1 if there is something, else in target:ip:proto:0
	// compare
	// expose
}
