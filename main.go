package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/jessevdk/go-flags"
	mp "github.com/mackerelio/go-mackerel-plugin"
	"github.com/mackerelio/golib/pluginutil"
	"github.com/prometheus/procfs"
)

// version by Makefile
var version string

type cmdOpts struct {
	Version                bool   `short:"v" long:"version" description:"Show version"`
	IgnoreInterfaces       string `long:"ignore-interfaces" description:"Regexp for interfaces name to ignore"`
	ignoreInterfacesRegexp *regexp.Regexp
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func generateTempfilePath() string {
	tmpDir := pluginutil.PluginWorkDir()
	curUser, _ := user.Current()
	uid := "0"
	if curUser != nil {
		uid = curUser.Uid
	}
	path := filepath.Join(tmpDir, fmt.Sprintf("mackerel-plugin-linux-netdev-%s", uid))
	return path
}

type stats struct {
	Interfaces map[string]procfs.NetDevLine `json:"interfaces"`
	Time       int64                        `json:"time"`
}

func writeStats(filename string, st map[string]procfs.NetDevLine) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	n := time.Now().Unix()
	jb, err := json.Marshal(stats{st, n})
	if err != nil {
		return err
	}
	_, err = file.Write(jb)
	return err
}

func readStats(filename string) (int64, map[string]procfs.NetDevLine, error) {
	st := stats{}
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, nil, err
	}
	err = json.Unmarshal(d, &st)
	if err != nil {
		return 0, nil, err
	}
	return st.Time, st.Interfaces, nil
}

type LinuxNetDevPlugin struct{}

func (u LinuxNetDevPlugin) GraphDefinition() map[string]mp.Graphs {
	return map[string]mp.Graphs{
		"linux-netdev.errors.#": {
			Label: "Linux NetDev errors per sec",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "tx", Label: "transmit errors encountered", Stacked: false},
				{Name: "rx", Label: "receive errors encountered", Stacked: false},
			},
		},
		"linux-netdev.dropped.#": {
			Label: "Linux NetDev dropped packets per sec",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "tx", Label: "packets dropped while transmitting", Stacked: false},
				{Name: "rx", Label: "packets dropped while receiving", Stacked: false},
			},
		},
		"linux-netdev.pps.#": {
			Label: "Linux NetDev packets per sec",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "tx", Label: "packets transmitted", Stacked: false},
				{Name: "rx", Label: "packets received", Stacked: false},
			},
		},
	}
}

func (u LinuxNetDevPlugin) FetchMetrics() (map[string]float64, error) {
	res := map[string]float64{}
	pfs, err := procfs.NewDefaultFS()
	if err != nil {
		return res, err
	}
	netdev, err := pfs.NetDev()
	if err != nil {
		return res, err
	}
	cur := map[string]procfs.NetDevLine{}
	for _, i := range netdev {
		if i.Name == "lo" {
			continue
		}
		if opts.IgnoreInterfaces != "" && opts.ignoreInterfacesRegexp.MatchString(i.Name) {
			continue
		}
		cur[i.Name] = i
	}

	path := generateTempfilePath()

	if !fileExists(path) {
		err = writeStats(path, cur)
		if err != nil {
			return res, err
		}
		return res, nil
	}

	defer func() {
		err := writeStats(path, cur)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}()

	t, prev, err := readStats(path)
	if err != nil {
		return res, err
	}
	if t == 0 {
		return res, fmt.Errorf("failed to get previous time")
	}
	n := time.Now().Unix()
	timeDiff := float64(n - t)
	if timeDiff > 600 {
		return res, fmt.Errorf("too long duration")
	}
	allTxErrors := float64(0)
	allTxDropped := float64(0)
	allRxErrors := float64(0)
	allRxDropped := float64(0)
	for k, c := range cur {
		p, ok := prev[k]
		if !ok {
			continue
		}
		var val float64
		// tx_errors
		val = float64(c.TxErrors - p.TxErrors)
		if val < 0 {
			val = 0
		}
		res[fmt.Sprintf("linux-netdev.errors.%s.tx", c.Name)] = val / timeDiff
		allTxErrors += val
		// tx_dropped
		val = float64(c.TxDropped - p.TxDropped)
		if val < 0 {
			val = 0
		}
		res[fmt.Sprintf("linux-netdev.dropped.%s.tx", c.Name)] = val / timeDiff
		allTxDropped += val
		// rx_errors
		val = float64(c.RxErrors - p.RxErrors)
		if val < 0 {
			val = 0
		}
		res[fmt.Sprintf("linux-netdev.errors.%s.rx", c.Name)] = val / timeDiff
		allRxErrors += val
		// rx_dropped
		val = float64(c.RxDropped - p.RxDropped)
		if val < 0 {
			val = 0
		}
		res[fmt.Sprintf("linux-netdev.dropped.%s.rx", c.Name)] = val / timeDiff
		allRxDropped += val
		// tx_packets
		val = float64(c.TxPackets - p.TxPackets)
		if val < 0 {
			val = 0
		}
		res[fmt.Sprintf("linux-netdev.pps.%s.tx", c.Name)] = val / timeDiff
		// tx_packets
		val = float64(c.RxPackets - p.RxPackets)
		if val < 0 {
			val = 0
		}
		res[fmt.Sprintf("linux-netdev.pps.%s.rx", c.Name)] = val / timeDiff
	}

	res["linux-netdev.errors.all.tx"] = allTxErrors / timeDiff
	res["linux-netdev.dropped.all.tx"] = allTxDropped / timeDiff
	res["linux-netdev.errors.all.rx"] = allRxErrors / timeDiff
	res["linux-netdev.dropped.all.rx"] = allRxDropped / timeDiff

	return res, nil
}

func main() {
	os.Exit(_main())
}

var opts = cmdOpts{}

func _main() int {
	psr := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	_, err := psr.Parse()
	if opts.Version {
		fmt.Printf(`%s %s
Compiler: %s %s
`,
			os.Args[0],
			version,
			runtime.Compiler,
			runtime.Version())
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	if opts.IgnoreInterfaces != "" {
		opts.ignoreInterfacesRegexp = regexp.MustCompile(opts.IgnoreInterfaces)
	}

	u := LinuxNetDevPlugin{}
	plugin := mp.NewMackerelPlugin(u)
	plugin.Run()
	return 0
}
