package scan

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// getDuration transforms a protocol's period into a time.Duration value.
func getDuration(period string) (time.Duration, error) {
	// only hours, minutes and seconds are handled by ParseDuration
	if strings.ContainsAny(period, "hms") {
		t, err := time.ParseDuration(period)
		if err != nil {
			return 0, err
		}
		return t, nil
	}

	sep := strings.Split(period, "d")
	days, err := strconv.Atoi(sep[0])
	if err != nil {
		return 0, err
	}

	t := time.Duration(days) * time.Hour * 24
	return t, nil
}

// readPortsRange transforms a range of ports given in conf to an array of
// effective ports
func readPortsRange(ranges string) ([]int, error) {
	ports := []int{}

	parts := strings.Split(ranges, ",")

	for _, spec := range parts {
		if spec == "" {
			continue
		}
		switch spec {
		case "all":
			for port := 1; port <= 65535; port++ {
				ports = append(ports, port)
			}
		case "reserved":
			for port := 1; port < 1024; port++ {
				ports = append(ports, port)
			}
		default:
			var decomposedRange []string

			if !strings.Contains(spec, "-") {
				decomposedRange = []string{spec, spec}
			} else {
				decomposedRange = strings.Split(spec, "-")
			}

			min, err := strconv.Atoi(decomposedRange[0])
			if err != nil {
				return nil, err
			}
			max, err := strconv.Atoi(decomposedRange[len(decomposedRange)-1])
			if err != nil {
				return nil, err
			}

			if min > max {
				return nil, fmt.Errorf("lower port %d is higher than high port %d", min, max)
			}
			if max > 65535 {
				return nil, fmt.Errorf("port %d is higher than max port", max)
			}
			for i := min; i <= max; i++ {
				ports = append(ports, i)
			}
		}
	}

	return ports, nil
}
