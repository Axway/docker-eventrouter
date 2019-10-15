# Event router LoadBalancer

`qlt-router` is a very *experimental* tool to evaluate the viability of dispatching Sentinel events accross several Sentinel frontend/acquisition servers.

ST/CFT/....   --> EventRouter --> qlt-router --> sentinel1, sentinel2, ...

## Usage

```bash
    ./qlt-router --qlt_port=3333 --sentinel_addrs=sentinel1:3333, sentinel2:3333,.... --sentinel_connections=1
```

`--sentinel_connections` : is the number of connections per sentinel server. More than one *may* speedup acquisition

## Caveats

- `qlt-router` may lose messages as acknowlegment are done too quickly (to be fixed)
- if multiple instances of `qlt-router` are used, they should have the same parameters in particular `--sentinel_addrs` and `--sentinel_connections`
- the first sentinel server will receive all non XFBTransfer messages (cyclelink, xfblogs...) (to be fixed)
- linux only for now
