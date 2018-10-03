package promlibvirt

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"sdstack.com/sdstack/compute"

	"github.com/prometheus/client_golang/prometheus"
	libvirt_plain "github.com/sdstack/go-libvirt-plain"
	libvirt_dbus "github.com/sdstack/go-libvirt-dbus"
)

var (
	// exporter metrics
	libvirtUpDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "", "up"),
		"Whether scraping libvirt's metrics was successful.",
		nil,
		nil)

	// memory metrics
	libvirtDomainMemoryMaximumDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory", "maximum_bytes"),
		"Maximum allowed memory of the domain, in bytes.",
		[]string{"domain"},
		nil)
	libvirtDomainMemoryCurrentDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory", "current_bytes"),
		"Memory usage of the domain, in bytes.",
		[]string{"domain"},
		nil)
	libvirtDomainMemoryResidentDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory", "resident_bytes"),
		"Memory usage of the domain, in bytes.",
		[]string{"domain"},
		nil)
	libvirtDomainMemoryLastUpdateDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_memory", "last_update_stats"),
		"Last update of memory stats, in seconds.",
		[]string{"domain"},
		nil)

	// cpu metrics
	libvirtDomainCpuTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_cpu", "time_seconds_total"),
		"Amount of CPU time used by the domain, in seconds.",
		[]string{"domain"},
		nil)
	libvirtDomainCpuSystemDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_cpu", "system_seconds_total"),
		"Amount of CPU time used by the domain, in seconds.",
		[]string{"domain"},
		nil)
	libvirtDomainCpuUserDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_cpu", "user_seconds_total"),
		"Amount of CPU time used by the domain, in seconds.",
		[]string{"domain"},
		nil)

	// vcpu metrics
	libvirtDomainVcpuMaximumDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_vcpu", "maximum"),
		"Number of maximum virtual CPUs for the domain.",
		[]string{"domain"},
		nil)
	libvirtDomainVcpuCurrentDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_vcpu", "current"),
		"Number of current virtual CPUs for the domain.",
		[]string{"domain"},
		nil)

	// block metrics
	libvirtDomainBlockAllocBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "alloc_bytes_total"),
		"Number of bytes allocated for device, in bytes.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainBlockCapBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "cap_bytes_total"),
		"Number of bytes capacity for device, in bytes.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainBlockPhysBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "phy_bytes_total"),
		"Number of bytes physical for device, in bytes.",
		[]string{"domain", "source", "target"},
		nil)

	libvirtDomainBlockRdBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "read_bytes_total"),
		"Number of bytes read from device, in bytes.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainBlockRdReqsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "read_requests_total"),
		"Number of read requests from device.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainBlockRdTimesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "read_seconds_total"),
		"Amount of time spent reading from a block device, in seconds.",
		[]string{"domain", "source", "target"},
		nil)

	libvirtDomainBlockWrBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "write_bytes_total"),
		"Number of bytes written from a block device, in bytes.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainBlockWrReqsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "write_requests_total"),
		"Number of write requests from a block device.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainBlockWrTimesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "write_seconds_total"),
		"Amount of time spent writing from a block device, in seconds.",
		[]string{"domain", "source", "target"},
		nil)

	libvirtDomainBlockFlReqsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "flush_requests_total"),
		"Number of flush requests from a block device.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainBlockFlTimesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_block", "flush_seconds_total"),
		"Amount of time spent flushing of a block device, in seconds.",
		[]string{"domain", "source", "target"},
		nil)

	// network metrcis
	libvirtDomainNetRxBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_net", "receive_bytes_total"),
		"Number of bytes received on a network interface, in bytes.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainNetRxPktsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_net", "receive_packets_total"),
		"Number of packets received on a network interface.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainNetRxErrsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_net", "receive_errors_total"),
		"Number of packet receive errors on a network interface.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainNetRxDropDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_net", "receive_drops_total"),
		"Number of packet receive drops on a network interface.",
		[]string{"domain", "source", "target"},
		nil)

	libvirtDomainNetTxBytesDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_net", "transmit_bytes_total"),
		"Number of bytes transmitted on a network interface, in bytes.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainNetTxPktsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_net", "transmit_packets_total"),
		"Number of packets transmitted on a network interface.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainNetTxErrsDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_net", "transmit_errors_total"),
		"Number of packet transmit errors on a network interface.",
		[]string{"domain", "source", "target"},
		nil)
	libvirtDomainNetTxDropDesc = prometheus.NewDesc(
		prometheus.BuildFQName("libvirt", "domain_net", "transmit_drops_total"),
		"Number of packet transmit drops on a network interface.",
		[]string{"domain", "source", "target"},
		nil)
)

// CollectDomain extracts Prometheus metrics from a libvirt domain.
func CollectStats(ch chan<- prometheus.Metric, stats map[string]map[string]interface{}) error {

	for dname, items := range stats {
		for ikey, ival := range items {
			idx := strings.Index(ikey, ".")
			idxl := strings.LastIndex(ikey, ".")
			key := ikey[:idx]
			switch key {
			default:
				fmt.Printf("zz %s %s %s\n", dname, ikey, ival)
				/*
									case "block":
									case "net":
									case "cpu":
							case "block":
					ch <- prometheus.MustNewConstMetric(
										libvirtDomainVcpuCurrentDesc,
										prometheus.GaugeValue,
										float64(ival.(uint32)),
										dname,  items[ikey[:idxl]+".path", items[ikey[:idxl]+".name")
				*/
			case "vcpu":
				switch ikey[idx+1:] {
				default:
					fmt.Printf("xx %s %s %s\n", dname, ikey, ival)
				case "maximum":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainVcpuMaximumDesc,
						prometheus.GaugeValue,
						float64(ival.(uint32)),
						dname)
				case "current":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainVcpuCurrentDesc,
						prometheus.GaugeValue,
						float64(ival.(uint32)),
						dname)
				}

			case "cpu":
				switch ikey[idx+1:] {
				case "user":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainCpuUserDesc,
						prometheus.CounterValue,
						float64(ival.(uint64))/1e9,
						dname)
				case "system":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainCpuSystemDesc,
						prometheus.CounterValue,
						float64(ival.(uint64))/1e9,
						dname)
				case "time":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainCpuTimeDesc,
						prometheus.CounterValue,
						float64(ival.(uint64))/1e9,
						dname)
				}
			case "balloon":
				switch ikey[idx+1:] {
				default:
					panic(fmt.Sprintf("xxx %s %s %s\n", dname, ikey, ival))
				case "last-update":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainMemoryLastUpdateDesc,
						prometheus.GaugeValue,
						float64(ival.(uint64)),
						dname)
				case "maximum":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainMemoryMaximumDesc,
						prometheus.GaugeValue,
						float64(ival.(uint64))*1024,
						dname)
				case "current":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainMemoryCurrentDesc,
						prometheus.GaugeValue,
						float64(ival.(uint64))*1024,
						dname)
				case "rss":
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainMemoryResidentDesc,
						prometheus.GaugeValue,
						float64(ival.(uint64))*1024,
						dname)
				}
			}
			/*
							if blockStats.RdBytesSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainBlockRdBytesDesc,
						prometheus.CounterValue,
						float64(blockStats.RdBytes),
						domainName,
						disk.Source.File.File,
						disk.Target.Dev)
				}

				if blockStats.RdReqSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainBlockRdReqDesc,
						prometheus.CounterValue,
						float64(blockStats.RdReq),
						domainName,
						disk.Source.File.File,
						disk.Target.Dev)
				}
				if blockStats.RdTotalTimesSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainBlockRdTotalTimesDesc,
						prometheus.CounterValue,
						float64(blockStats.RdTotalTimes)/1e9,
						domainName,
						disk.Source.File.File,
						disk.Target.Dev)
				}
				if blockStats.WrBytesSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainBlockWrBytesDesc,
						prometheus.CounterValue,
						float64(blockStats.WrBytes),
						domainName,
						disk.Source.File.File,
						disk.Target.Dev)
				}
				if blockStats.WrReqSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainBlockWrReqDesc,
						prometheus.CounterValue,
						float64(blockStats.WrReq),
						domainName,
						disk.Source.File.File,
						disk.Target.Dev)
				}
				if blockStats.WrTotalTimesSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainBlockWrTotalTimesDesc,
						prometheus.CounterValue,
						float64(blockStats.WrTotalTimes)/1e9,
						domainName,
						disk.Source.File.File,
						disk.Target.Dev)
				}
				if blockStats.FlushReqSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainBlockFlushReqDesc,
						prometheus.CounterValue,
						float64(blockStats.FlushReq),
						domainName,
						disk.Source.File.File,
						disk.Target.Dev)
				}
				if blockStats.FlushTotalTimesSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainBlockFlushTotalTimesDesc,
						prometheus.CounterValue,
						float64(blockStats.FlushTotalTimes)/1e9,
						domainName,
						disk.Source.File.File,
						disk.Target.Dev)
				}

				if interfaceStats.RxBytesSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainInterfaceRxBytesDesc,
						prometheus.CounterValue,
						float64(interfaceStats.RxBytes),
						domainName,
						iface.Source.Bridge,
						iface.Target.Dev)
				}
				if interfaceStats.RxPacketsSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainInterfaceRxPacketsDesc,
						prometheus.CounterValue,
						float64(interfaceStats.RxPackets),
						domainName,
						iface.Source.Bridge,
						iface.Target.Dev)
				}
				if interfaceStats.RxErrsSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainInterfaceRxErrsDesc,
						prometheus.CounterValue,
						float64(interfaceStats.RxErrs),
						domainName,
						iface.Source.Bridge,
						iface.Target.Dev)
				}
				if interfaceStats.RxDropSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainInterfaceRxDropDesc,
						prometheus.CounterValue,
						float64(interfaceStats.RxDrop),
						domainName,
						iface.Source.Bridge,
						iface.Target.Dev)
				}
				if interfaceStats.TxBytesSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainInterfaceTxBytesDesc,
						prometheus.CounterValue,
						float64(interfaceStats.TxBytes),
						domainName,
						iface.Source.Bridge,
						iface.Target.Device)
				}
				if interfaceStats.TxPacketsSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainInterfaceTxPacketsDesc,
						prometheus.CounterValue,
						float64(interfaceStats.TxPackets),
						domainName,
						iface.Source.Bridge,
						iface.Target.Device)
				}
				if interfaceStats.TxErrsSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainInterfaceTxErrsDesc,
						prometheus.CounterValue,
						float64(interfaceStats.TxErrs),
						domainName,
						iface.Source.Bridge,
						iface.Target.Device)
				}
				if interfaceStats.TxDropSet {
					ch <- prometheus.MustNewConstMetric(
						libvirtDomainInterfaceTxDropDesc,
						prometheus.CounterValue,
						float64(interfaceStats.TxDrop),
						domainName,
						iface.Source.Bridge,
						iface.Target.Device)
				}
			*/
		}
	}

	return nil
}

// CollectFromLibvirt obtains Prometheus metrics from all domains in a
// libvirt setup.
func CollectFromLibvirt(ch chan<- prometheus.Metric, uri string) error {
	var err error
	var hyper string
	var driver string
	var proto string
	var stats map[string]map[string]interface{}

	u, err := url.Parse(uri)
	if err != nil {
		return err
	}
// qemu+tcp native
// plain+qemu+tcp

	fields := strings.Fields(u.Scheme, "+")
  switch len(fields) {
	case 3:
		driver = fields[0]
		hyper = fields[1]
		proto = fields[2]
	case 2:
		driver = "auto"
		hyper = fields[0]
		proto = fields[1]
	default:
		driver = "auto"
		hyper = 
	}

	switch u.Scheme {
	default:
		return fmt.Errorf("invalid driver: %s", u.Scheme)
	case "dbus":
		if idx > 0 {
		
		}
		lv := libvirt_dbus.NewConn(DriverQEMU)
		err := lv.Connect("")
		if err != nil {
			return err
		}
		stats, err = lv.ConnectGetAllDomainStats(0, 0) //536870912
		if err != nil {
			return err
		}
	}

	return CollectStats(ch, stats)
}

// LibvirtExporter implements a Prometheus exporter for libvirt state.
type LibvirtExporter struct {
	uri net.URL
}

// Describe returns metadata for all Prometheus metrics that may be exported.
func (e *LibvirtExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- libvirtUpDesc

	ch <- libvirtDomainMemoryMaximumDesc
	ch <- libvirtDomainMemoryCurrentDesc
	ch <- libvirtDomainMemoryResidentDesc
	ch <- libvirtDomainMemoryLastUpdateDesc

	ch <- libvirtDomainCpuTimeDesc
	ch <- libvirtDomainCpuSystemDesc
	ch <- libvirtDomainCpuUserDesc

	ch <- libvirtDomainVcpuMaximumDesc
	ch <- libvirtDomainVcpuCurrentDesc

	ch <- libvirtDomainBlockRdBytesDesc
	ch <- libvirtDomainBlockRdReqsDesc
	ch <- libvirtDomainBlockRdTimesDesc
	ch <- libvirtDomainBlockWrBytesDesc
	ch <- libvirtDomainBlockWrReqsDesc
	ch <- libvirtDomainBlockWrTimesDesc
	ch <- libvirtDomainBlockFlReqsDesc
	ch <- libvirtDomainBlockFlTimesDesc
}

// Collect scrapes Prometheus metrics from libvirt.
func (e *LibvirtExporter) Collect(ch chan<- prometheus.Metric) {
	err := CollectFromLibvirt(ch, e.uri)
	if err == nil {
		ch <- prometheus.MustNewConstMetric(
			libvirtUpDesc,
			prometheus.GaugeValue,
			1.0)
	} else {
		log.Printf("Failed to scrape metrics: %s", err)
		ch <- prometheus.MustNewConstMetric(
			libvirtUpDesc,
			prometheus.GaugeValue,
			0.0)
	}
}


func NewLibvirtExporter(uri string) (*LibvirtExporter, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	return &LibvirtExporter{uri: u}, nil
}
