package collectors

import (
	"fmt"
	"sync"
	"time"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/net"
	"github.com/ovh/noderig/core"
)

// Net collects network related metrics
type Net struct {
	interfaces []string
	mutex      sync.RWMutex
	MetricsCollected   []core.MetricCollected
	level      uint8
	period     uint
}

// NewNet returns an initialized Net collector.
func NewNet(period uint, level uint8, opts interface{}) *Net {

	var ifaces []string

	if opts != nil {
		if options, ok := opts.(map[string]interface{}); ok {
			if val, ok := options["interfaces"]; ok {
				if ifs, ok := val.([]interface{}); ok {
					for _, v := range ifs {
						if s, ok := v.(string); ok {
							ifaces = append(ifaces, s)
						}
					}
				}
			}
		}

	}

	c := &Net{
		level:      level,
		period:     period,
		interfaces: ifaces,
	}

	if level == 0 {
		return c
	}

	tick := time.Tick(time.Duration(period) * time.Millisecond)
	go func() {
		for range tick {
			if err := c.scrape(); err != nil {
				log.Error(err)
			}
		}
	}()

	return c
}

// Metrics delivers metrics.
func (c *Net) Metrics() []core.MetricCollected {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.MetricsCollected
}

func (c *Net) scrape() error {
	counters, err := net.IOCounters(true)
	if err != nil {
		return err
	}

	var in, out uint64
	for _, cnt := range counters {
		if cnt.Name == "lo" {
			continue
		} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
			continue
		}
		in += cnt.BytesRecv
		out += cnt.BytesSent
	}
	in = in / uint64(c.period/1000)
	out = out / uint64(c.period/1000)

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.MetricsCollected = nil

	// timestamp := time.Now().UnixNano()/1000

	if c.level == 1 {
		gts := fmt.Sprintf("os.net.bytes{direction=in}")
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatUint(in, 10),
		})

		gts = fmt.Sprintf("os.net.bytes{direction=out}")
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatUint(out, 10),
		})
	}

	if c.level > 1 {
		for _, cnt := range counters {
			if cnt.Name == "lo" {
				continue
			} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
				continue
			}
			gts := fmt.Sprintf("os.net.bytes{iface=%v,direction=in}", cnt.Name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(cnt.BytesRecv, 10),
			})

			gts = fmt.Sprintf("os.net.bytes{iface=%v,direction=out}", cnt.Name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(cnt.BytesSent, 10),
			})
		}
	}

	if c.level > 2 {
		for _, cnt := range counters {
			if cnt.Name == "lo" {
				continue
			} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
				continue
			}
			gts := fmt.Sprintf("os.net.packets{iface=%v,direction=in}", cnt.Name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(cnt.PacketsRecv, 10),
			})

			gts = fmt.Sprintf("os.net.packets{iface=%v,direction=out}", cnt.Name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(cnt.PacketsSent, 10),
			})

			gts = fmt.Sprintf("os.net.errs{iface=%v,direction=in}", cnt.Name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(cnt.Errin, 10),
			})

			gts = fmt.Sprintf("os.net.errs{iface=%v,direction=out}", cnt.Name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(cnt.Errout, 10),
			})

			gts = fmt.Sprintf("os.net.dropped{iface=%v,direction=in}", cnt.Name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(cnt.Dropin, 10),
			})

			gts = fmt.Sprintf("os.net.dropped{iface=%v,direction=out}", cnt.Name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(cnt.Dropout, 10),
			})
		}
	}

	return nil
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
