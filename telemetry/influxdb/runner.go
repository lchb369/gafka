package influxdb

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/funkygao/gafka/telemetry"
	"github.com/funkygao/go-metrics"
	"github.com/influxdata/influxdb/client"
)

var _ telemetry.Reporter = &runner{}

type runner struct {
	cf     *config
	reg    metrics.Registry
	client *client.Client

	quiting, quit chan struct{}
}

// New creates a InfluxDB reporter which will post the metrics from the given registry at each interval.
// CREATE RETENTION POLICY two_hours ON food_data DURATION 2h REPLICATION 1 DEFAULT
// SHOW RETENTION POLICIES ON food_data
// CREATE CONTINUOUS QUERY cq_30m ON food_data BEGIN SELECT mean(website) AS mean_website,mean(phone) AS mean_phone INTO food_data."default".downsampled_orders FROM orders GROUP BY time(30m) END
func New(r metrics.Registry, cf *config) telemetry.Reporter {
	this := &runner{
		reg:     r,
		cf:      cf,
		quiting: make(chan struct{}),
		quit:    make(chan struct{}),
	}

	return this
}

func (this *runner) makeClient() (err error) {
	this.client, err = client.NewClient(client.Config{
		URL:      this.cf.url,
		Username: this.cf.username,
		Password: this.cf.password,
		Timeout:  time.Second * 4,
	})

	_, _, err = this.client.Ping()
	if err != nil {
		this.client = nil // to trigger retry
	}

	return
}

func (*runner) Name() string {
	return "influxdb"
}

func (this *runner) Stop() {
	close(this.quiting)
	<-this.quit
}

func (this *runner) Start() error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
	}()

	intervalTicker := time.Tick(this.cf.interval)
	pts := make([]client.Point, 0, 1<<8)
	for {
		select {
		case <-this.quiting:
			// drain
			pts = pts[:0]
			this.export(&pts)
			this.dump(pts)

			close(this.quit)
			return nil

		case <-intervalTicker:
			pts = pts[:0]
			this.export(&pts)
			this.dump(pts)

		}
	}

	return nil
}
