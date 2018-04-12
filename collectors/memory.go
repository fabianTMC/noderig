package collectors

import (
	"fmt"
	"sync"
	"time"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/mem"
	"github.com/ovh/noderig/core"
)

// Memory collects memory related metrics
type Memory struct {
	mutex     sync.RWMutex
	MetricsCollected   []core.MetricCollected
	level     uint8
}

// NewMemory returns an initialized Memory collector.
func NewMemory(period uint, level uint8) *Memory {
	c := &Memory{
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
func (c *Memory) Metrics() []core.MetricCollected {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.MetricsCollected
}

func (c *Memory) scrape() error {
	virt, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		return err
	}

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Delete previous metrics
	c.MetricsCollected = nil

	// timestamp := time.Now().UnixNano()/1000

	gts := fmt.Sprintf("os.mem{}")
	c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
		Name: gts,
		Value: strconv.FormatFloat(virt.UsedPercent, 'f', 6, 64),
	})

	gts = fmt.Sprintf("os.swap{}")
	c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
		Name: gts,
		Value: strconv.FormatFloat(swap.UsedPercent, 'f', 6, 64),
	})

	if c.level > 1 {
		gts := fmt.Sprintf("os.mem.used{}")
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatUint(virt.Used, 10),
		})
		gts = fmt.Sprintf("os.mem.total{}")
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatUint(virt.Total, 10),
		})
		gts = fmt.Sprintf("os.swap.used{}")
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatUint(swap.Used, 10),
		})
		gts = fmt.Sprintf("os.swap.total{}")
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatUint(swap.Total, 10),
		})
	}

	return nil
}
