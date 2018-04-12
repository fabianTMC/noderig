package collectors

import (
	"fmt"
	"sync"
	"time"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/cpu"
	"github.com/ovh/noderig/core"
)

// CPU collects cpu related metrics
type CPU struct {
	times []cpu.TimesStat

	mutex     sync.RWMutex
	MetricsCollected   []core.MetricCollected
	level     uint8
}

// NewCPU returns an initialized CPU collector.
func NewCPU(period uint, level uint8) *CPU {
	c := &CPU{
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
func (c *CPU) Metrics() []core.MetricCollected {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.MetricsCollected
}

// https://github.com/Leo-G/DevopsWiki/wiki/How-Linux-CPU-Usage-Time-and-Percentage-is-calculated
func (c *CPU) scrape() error {
	times, err := cpu.Times(true)
	if err != nil {
		return err
	}

	if len(c.times) == 0 { // init
		c.times = times
		return nil
	}

	idles := make([]float64, len(times))
	systems := make([]float64, len(times))
	users := make([]float64, len(times))
	iowaits := make([]float64, len(times))
	nices := make([]float64, len(times))
	irqs := make([]float64, len(times))
	for i, t := range times {
		dt := t.Total() - c.times[i].Total()
		idles[i] = (t.Idle - c.times[i].Idle) / dt
		systems[i] = (t.System - c.times[i].System) / dt
		users[i] = (t.User - c.times[i].User) / dt
		iowaits[i] = (t.Iowait - c.times[i].Iowait) / dt
		nices[i] = (t.Nice - c.times[i].Nice) / dt
		irqs[i] = (t.Irq - c.times[i].Irq) / dt
	}

	global := 0.0
	for _, v := range idles {
		global += v
	}
	global = (1.0 - global/float64(len(idles))) * 100

	c.times = times

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Delete previous metrics
	c.MetricsCollected = nil

	class := fmt.Sprintf("os.cpu")
	// timestamp := time.Now().UnixNano()/1000

	gts := fmt.Sprintf("%v{}", class)
	c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
		Name: gts,
		Value: strconv.FormatFloat(global, 'f', 6, 64),
	})

	if c.level == 2 {
		iowait := 0.0
		for _, v := range iowaits {
			iowait += v
		}
		iowait = iowait / float64(len(iowaits)) * 100
		gts := fmt.Sprintf("%v.iowait{}", class)
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatFloat(iowait, 'f', 6, 64),
		})

		user := 0.0
		for _, v := range users {
			user += v
		}
		user = user / float64(len(users)) * 100
		gts = fmt.Sprintf("%v.user{}", class)
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatFloat(user, 'f', 6, 64),
		})

		system := 0.0
		for _, v := range systems {
			system += v
		}
		system = system / float64(len(systems)) * 100
		gts = fmt.Sprintf("%v.systems{}", class)
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatFloat(system, 'f', 6, 64),
		})

		nice := 0.0
		for _, v := range nices {
			nice += v
		}
		nice = nice / float64(len(nices)) * 100
		gts = fmt.Sprintf("%v.nice{}", class)
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatFloat(nice, 'f', 6, 64),
		})

		irq := 0.0
		for _, v := range irqs {
			irq += v
		}
		irq = irq / float64(len(irqs)) * 100
		gts = fmt.Sprintf("%v.irq{}", class)
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatFloat(irq, 'f', 6, 64),
		})
	}

	if c.level == 3 {
		for i, v := range iowaits {
			gts := fmt.Sprintf("%v.iowait{chore=%v}", class, i)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatFloat(v*100, 'f', 6, 64),
			})
		}

		for i, v := range users {
			gts := fmt.Sprintf("%v.user{chore=%v}", class, i)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatFloat(v*100, 'f', 6, 64),
			})
		}

		for i, v := range systems {
			gts := fmt.Sprintf("%v.systems{chore=%v}", class, i)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatFloat(v*100, 'f', 6, 64),
			})
		}

		for i, v := range nices {
			gts := fmt.Sprintf("%v.nice{chore=%v}", class, i)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatFloat(v*100, 'f', 6, 64),
			})
		}

		for i, v := range irqs {
			gts := fmt.Sprintf("%v.irq{chore=%v}", class, i)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatFloat(v*100, 'f', 6, 64),
			})
		}
	}

	return nil
}
