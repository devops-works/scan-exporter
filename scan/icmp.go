package scan

import (
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func (t *target) ping(timeout time.Duration) {
	p, err := getDuration(t.icmpPeriod)
	if err != nil {
		log.Fatal().Err(err).Msgf("cannot parse duration %s", t.icmpPeriod)
	}

	// Randomize period to avoid listening override.
	// The random time added will be between 1 and 1.5s
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(500) + 1000
	randPeriod := p + (time.Duration(n) * time.Millisecond)

	ticker := time.NewTicker(randPeriod)

	for {
		select {
		case <-ticker.C:
			destAddr := &net.IPAddr{IP: net.ParseIP(t.ip)}
			c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
			if err != nil {
				log.Error().Err(err).Msg("error sending ping request")
			}
			defer c.Close()

			m := icmp.Message{
				Type: ipv4.ICMPTypeEcho,
				Code: 0,
				Body: &icmp.Echo{
					ID:   os.Getpid() & 0xffff,
					Data: []byte(""),
				},
			}

			data, err := m.Marshal(nil)
			if err != nil {
				log.Error().Err(err).Msg("error sending ping request")
			}

			start := time.Now()
			_, err = c.WriteTo(data, destAddr)
			if err != nil {
				log.Error().Err(err).Msg("error sending ping request")
			}

			reply := make([]byte, 1500)
			err = c.SetReadDeadline(time.Now().Add(timeout))
			if err != nil {
				log.Error().Err(err).Msg("error sending ping request")
			}
			n, SourceIP, err := c.ReadFrom(reply)
			// timeout
			if err != nil {
				log.Error().Err(err).Msg("error sending ping request")
			}
			// if anything is read from the connection it means that the host is alive
			if destAddr.String() == SourceIP.String() && n > 0 {

				rtt := time.Since(start)
				log.Info().Msgf("%s (%s) ICMP rtt: %s", t.name, t.ip, rtt)

				break
			}
			log.Warn().Msgf("%s (%s) does not respond to ICMP requests", t.name, t.ip)
		}
	}
}
