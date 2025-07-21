package scan

import (
	"fmt"
	"slices"
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

// readPortsRange transforms a comma-separated string of ports into a unique,
// sorted slice of integers.
func readPortsRange(ranges string) ([]int, error) {
	ports := []int{}

	// Remove spaces
	ranges = strings.ReplaceAll(ranges, " ", "")

	parts := strings.SplitSeq(ranges, ",")

	for spec := range parts {
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
		case "top1000":
			ports = append(ports, top1000Ports...)
		default:
			if strings.Contains(spec, "-") {
				decomposedRange := strings.Split(spec, "-")
				if len(decomposedRange) != 2 || decomposedRange[0] == "" || decomposedRange[1] == "" {
					return nil, fmt.Errorf("invalid port range format: %q", spec)
				}

				min, err := strconv.Atoi(decomposedRange[0])
				if err != nil {
					return nil, fmt.Errorf("invalid start port in range %q: %w", spec, err)
				}
				max, err := strconv.Atoi(decomposedRange[1])
				if err != nil {
					return nil, fmt.Errorf("invalid end port in range %q: %w", spec, err)
				}

				if min > max {
					return nil, fmt.Errorf("start port %d is higher than end port %d in range %q", min, max, spec)
				}

				if min < 1 || max > 65535 {
					return nil, fmt.Errorf("port range %q is out of the valid range (1-65535)", spec)
				}

				for i := min; i <= max; i++ {
					ports = append(ports, i)
				}
			} else {
				port, err := strconv.Atoi(spec)
				if err != nil {
					return nil, fmt.Errorf("invalid port specification %q: %w", spec, err)
				}

				if port < 1 || port > 65535 {
					return nil, fmt.Errorf("port %d is out of the valid range (1-65535)", port)
				}

				ports = append(ports, port)
			}
		}
	}

	slices.Sort(ports)
	uniquePorts := slices.Compact(ports)

	return uniquePorts, nil
}
