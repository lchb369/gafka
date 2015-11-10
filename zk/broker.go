package zk

import (
	"fmt"
	"time"
)

type Broker struct {
	JmxPort   int      `json:"jmx_port"`
	Timestamp string   `json:"timestamp"`
	Endpoints []string `json:"endpoints"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Version   int      `json:"version"`
}

func (b Broker) String() string {
	return fmt.Sprintf("%s:%d ver:%d uptime:%s",
		b.Host, b.Port,
		b.Version,
		time.Since(TimestampToTime(b.Timestamp)))
}
