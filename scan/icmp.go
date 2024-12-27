package scan

import (
	"math/rand"
	"time"

	"github.com/devops-works/scan-exporter/metrics"
	"github.com/go-ping/ping"
	"github.com/rs/zerolog"
)

// ping realises an ICMP echo request to a specified target.
// Each error is followed by a continue, which will not stop the goroutine.
func (t *target) ping(logger zerolog.Logger, timeout time.Duration, pchan chan metrics.PingInfo) {
	p, err := getDuration(t.icmpPeriod)
	if err != nil {
		logger.Fatal().Err(err).Msgf("cannot parse duration %s", t.icmpPeriod)
	}

	// Randomize period to avoid listening override.
	// The random time added will be between 1 and 1.5s
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(500) + 1000
	randPeriod := p + (time.Duration(n) * time.Millisecond)

	ticker := time.NewTicker(randPeriod)

	for range ticker.C {
		pinfo := metrics.PingInfo{
			Name:         t.name,
			IP:           t.ip,
			IsResponding: false,
			RTT:          0,
		}

		pinger, err := ping.NewPinger(t.ip)
		if err != nil {
			logger.Error().Err(err).Msgf("error creating pinger for %s (%s)", t.name, t.ip)
			continue
		}

		pinger.Timeout = timeout
		pinger.SetPrivileged(true)
		pinger.Count = 3

		pinger.OnFinish = func(stats *ping.Statistics) {
			logger.Debug().Str("name", t.name).Str("ip", t.ip).Msgf("ping ended")
			pinfo.RTT = stats.AvgRtt
			if stats.AvgRtt != 0 {
				pinfo.IsResponding = true
			} else {
				pinfo.IsResponding = false
			}
			pchan <- pinfo
		}

		pinger.OnRecv = func(p *ping.Packet) {
			logger.Debug().Str("name", t.name).Str("ip", t.ip).Msgf("received one ICMP reply")
		}

		logger.Debug().Str("name", t.name).Str("ip", t.ip).Msgf("running a new ping")
		err = pinger.Run()
		if err != nil {
			logger.Error().Err(err).Msgf("error running pinger for %s (%s)", t.name, t.ip)
			continue
		}
	}
}
