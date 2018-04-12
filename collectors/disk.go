package collectors

import (
	"fmt"
	"path"
	"sync"
	"time"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/disk"
	"github.com/fabianTMC/noderig/core"
)

// Disk collects disk related metrics
type Disk struct {
	mutex        sync.RWMutex
	MetricsCollected   []core.MetricCollected
	level        uint8
	period       uint
	allowedDisks map[string]struct{}
}

// NewDisk returns an initialized Disk collector.
func NewDisk(period uint, level uint8, opts interface{}) *Disk {

	allowedDisks := map[string]struct{}{}
	if opts != nil {
		if options, ok := opts.(map[string]interface{}); ok {
			if val, ok := options["names"]; ok {
				if diskNames, ok := val.([]interface{}); ok {
					for _, v := range diskNames {
						if diskName, ok := v.(string); ok {
							allowedDisks[diskName] = struct{}{}
						}
					}
				}
			}
		}
	}

	c := &Disk{
		level:        level,
		period:       period,
		allowedDisks: allowedDisks,
	}

	if level > 0 {
		tick := time.Tick(time.Duration(period) * time.Millisecond)
		go func() {
			for range tick {
				if err := c.scrape(); err != nil {
					log.Error(err)
				}
			}
		}()
	}

	return c
}

// Metrics delivers metrics.
func (c *Disk) Metrics() []core.MetricCollected {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.MetricsCollected
}

func (c *Disk) scrape() error {
	counters, err := disk.IOCounters()
	if err != nil {
		return err
	}

	parts, err := disk.Partitions(false)
	if err != nil {
		return err
	}

	dev := make(map[string]disk.UsageStat)
	for _, p := range parts {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		dev[p.Device] = *usage
	}

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.MetricsCollected = nil

	now := fmt.Sprintf("os.disk.fs")
	// timestamp := time.Now().UnixNano()/1000

	for diskPath, usage := range dev {
		if len(c.allowedDisks) > 0 {
			_, diskName := path.Split(diskPath) // return "sda1" from "/dev/sda1"
			if _, allowed := c.allowedDisks[diskName]; !allowed {
				log.Debug("Disk " + diskPath + " is blacklisted, skip it")
				continue
			}
		}
		gts := fmt.Sprintf("%v{disk=%v}{mount=%v}", now, diskPath, usage.Path)
		c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
			Name: gts,
			Value: strconv.FormatFloat(usage.UsedPercent, 'f', 6, 64),
		})
	}

	if c.level > 1 {
		for diskPath, usage := range dev {
			if len(c.allowedDisks) > 0 {
				_, diskName := path.Split(diskPath) // return "sda1" from "/dev/sda1"
				if _, allowed := c.allowedDisks[diskName]; !allowed {
					log.Debug("Disk " + diskPath + " is blacklisted, skip it")
					continue
				}
			}

			gts := fmt.Sprintf("%v.used{disk=%v}{mount=%v}", now, diskPath, usage.Path)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(usage.Used, 10),
			})
			gts = fmt.Sprintf("%v.total{disk=%v}{mount=%v}", now, diskPath, usage.Path)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(usage.Total, 10),
			})
			gts = fmt.Sprintf("%v.inodes.used{disk=%v}{mount=%v}", now, diskPath, usage.Path)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(usage.InodesUsed, 10),
			})
			gts = fmt.Sprintf("%v.inodes.total{disk=%v}{mount=%v}", now, diskPath, usage.Path)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(usage.InodesTotal, 10),
			})
		}
	}

	if c.level > 2 {
		for name, stats := range counters {
			if len(c.allowedDisks) > 0 {
				if _, allowed := c.allowedDisks[name]; !allowed {
					log.Debug("Disk name " + name + " is blacklisted, skip it")
					continue
				}
			}

			gts := fmt.Sprintf("%v.bytes.read{name=%v}", now, name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(stats.ReadBytes, 10),
			})

			gts = fmt.Sprintf("%v.bytes.write{name=%v}", now, name)
			c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
				Name: gts,
				Value: strconv.FormatUint(stats.WriteBytes, 10),
			})

			if c.level > 3 {
				gts = fmt.Sprintf("%v.io.read{name=%v}", now, name)
				c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
					Name: gts,
					Value: strconv.FormatUint(stats.ReadCount, 10),
				})
				gts = fmt.Sprintf("%v.io.write{name=%v}", now, name)
				c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
					Name: gts,
					Value: strconv.FormatUint(stats.WriteCount, 10),
				})

				if c.level > 4 {
					gts = fmt.Sprintf("%v.io.read.ms{name=%v}", now, name)
					c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
						Name: gts,
						Value: strconv.FormatUint(stats.ReadTime, 10),
					})
					gts = fmt.Sprintf("%v.io.write.ms{name=%v}", now, name)
					c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
						Name: gts,
						Value: strconv.FormatUint(stats.WriteTime, 10),
					})
					gts = fmt.Sprintf("%v.io{name=%v}", now, name)
					c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
						Name: gts,
						Value: strconv.FormatUint(stats.IopsInProgress, 10),
					})
					gts = fmt.Sprintf("%v.io.ms{name=%v}", now, name)
					c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
						Name: gts,
						Value: strconv.FormatUint(stats.IoTime, 10),
					})
					gts = fmt.Sprintf("%v.io.weighted.ms{name=%v}", now, name)
					c.MetricsCollected = append(c.MetricsCollected, core.MetricCollected{
						Name: gts,
						Value: strconv.FormatUint(stats.WeightedIO, 10),
					})
				}
			}
		}
	}

	return nil
}
