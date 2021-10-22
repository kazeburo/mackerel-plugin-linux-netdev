# mackerel-plugin-linux-netdev

mackerel metric plugin for linux proc/net/dev. collect errors, droppped and packets per sec.

## Usage

```
Usage:
  mackerel-plugin-linux-netdev [OPTIONS]

Application Options:
  -v, --version            Show version
      --ignore-interfaces= Regexp for interfaces name to ignore

Help Options:
  -h, --help               Show this help message
```

```
$ ./mackerel-plugin-linux-netdev
linux-netdev.errors.all.tx      0       1634886461
linux-netdev.errors.eth0.tx     0       1634886461
linux-netdev.errors.eth0.rx     0       1634886461
linux-netdev.errors.all.rx      0       1634886461
linux-netdev.dropped.eth0.tx    0       1634886461
linux-netdev.dropped.all.tx     0       1634886461
linux-netdev.dropped.eth0.rx    0       1634886461
linux-netdev.dropped.all.rx     0       1634886461
linux-netdev.pps.eth0.tx        1.666667        1634886461
linux-netdev.pps.eth0.rx        1529.333333     1634886461
```

## Install

Please download release page or `mkr plugin install kazeburo/mackerel-plugin-linux-netdev`.