package core

type MetricCollected struct {
	Name string `json:"name"`
	Value string `json:"value"`
}

// Collector interface
type Collector interface {
	Metrics() []MetricCollected
}
