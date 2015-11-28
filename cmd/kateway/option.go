package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	options struct {
		zone                 string
		cluster              string
		configFile           string
		logFile              string
		logLevel             string
		mode                 string
		crashLogFile         string
		influxServer         string
		cpuprof              bool
		showVersion          bool
		memprof              bool
		blockprof            bool
		port                 int
		maxPubSize           int64
		maxClients           int
		offsetCommitInterval time.Duration
		reporterInterval     time.Duration
		metaRefresh          time.Duration
		httpReadTimeout      time.Duration
		httpWriteTimeout     time.Duration
	}
)

func parseFlags() {
	flag.StringVar(&options.zone, "zone", "", "kafka zone name")
	flag.StringVar(&options.cluster, "cluster", "", "kafka cluster name")
	flag.DurationVar(&options.metaRefresh, "metarefresh", time.Minute, "meta data refresh interval")
	flag.IntVar(&options.port, "port", 9090, "http bind port")
	flag.StringVar(&options.logLevel, "level", "debug", "log level")
	flag.StringVar(&options.logFile, "log", "stdout", "log file, default stdout")
	flag.StringVar(&options.crashLogFile, "crashlog", "", "crash log")
	flag.StringVar(&options.configFile, "conf", "/etc/gafka.cf", "config file")
	flag.BoolVar(&options.showVersion, "version", false, "show version and exit")
	flag.DurationVar(&options.reporterInterval, "report", time.Second*10, "reporter flush interval")
	flag.BoolVar(&options.cpuprof, "cpuprof", false, "enable cpu profiling")
	flag.BoolVar(&options.memprof, "memprof", false, "enable memory profiling")
	flag.StringVar(&options.mode, "mode", "pub", "gateway mode: <pub|sub|pubsub>")
	flag.StringVar(&options.influxServer, "influxdb", "http://10.77.144.193:10036", "influxdb server address for the metrics reporter")
	flag.BoolVar(&options.blockprof, "blockprof", false, "enable block profiling")
	flag.Int64Var(&options.maxPubSize, "maxpub", 1<<20, "max Pub message size")
	flag.IntVar(&options.maxClients, "maxclient", 10000, "max concurrent connections")
	flag.DurationVar(&options.offsetCommitInterval, "offsetcommit", time.Minute, "consumer offset commit interval")
	flag.DurationVar(&options.httpReadTimeout, "httprtimeout", time.Second*60, "http server read timeout")
	flag.DurationVar(&options.httpWriteTimeout, "httpwtimeout", time.Second*60, "http server write timeout")

	flag.Parse()
}

func validateFlags() {
	if options.zone == "" || options.cluster == "" {
		fmt.Fprintf(os.Stderr, "-zone and -cluster are required\n")
		os.Exit(1)
	}
}
