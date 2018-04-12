package collectors

import (
	"fmt"
	"sync"
	"time"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/load"
	"github.com/fabianTMC/noderig/core"
)

// Load collects load related metrics
type Load struct {
	mutex     sync.RWMutex
	MetricsCollected   []core.MetricCollected
	level     uint8
}

// NewLoad returns an initialized Load collector.
func NewLoad(period uint, level uint8) *Load {
	c := &Load{
		level: level,
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
func (c *Load) Metrics() []core.MetricCollected {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.MetricsCollected
}

func (c *Load) scrape() error {
	avg, err := load.Avg()
	if err != nil {
		return err
	}

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Delete previous metrics
	c.MetricsCollected = nil

	// timestamp :=  time.Now().UnixNano()/1000

	gts := fmt.Sprintf("os.load1{}")
	c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
		Name: gts,
		Value: strconv.FormatFloat(avg.Load1, 'f', 6, 64),
	})

	if c.level > 1 {
		gts := fmt.Sprintf("os.load5{}")
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatFloat(avg.Load5, 'f', 6, 64),
		})

		gts = fmt.Sprintf("os.load15{}")
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatFloat(avg.Load15, 'f', 6, 64),
		})
	}

	return nil
}
