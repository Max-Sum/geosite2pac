# Geosite2PAC

[![Docker](https://github.com/Max-Sum/geosite2pac/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/Max-Sum/geosite2pac/actions/workflows/docker-publish.yml)

This tool generates a Proxy Auto-Configuration(PAC) file from the well-established geosite.dat file of v2ray.

## Run

Generate:

```
geosite2pac -rule config/rule.json -output autoproxy.pac
```

Serve as WPAD file server

```
geosite2pac -rule config/rule.json -listen :80
```

or you can run from docker:

```
docker run --name geosite2pac -p 80:8000 -v config:/app/config gzmaxsum/geosite2pac
```

#### Reload & purge cache

When the program serves as WPAD file server. It caches the generated pac for 24 hours. To purge the cache, send SIGHUP to the process:

```
kill -HUP $(pidof geosite2pac)
# or send to docker container
docker kill geosite2pac --signal=SIGHUP
```

## Configuration

rule.json is the main configuration file, contains an map. The key is very similar to v2ray's [RuleObject-domains](https://www.v2fly.org/config/routing.html#routingobject), with some specialties:

- `ext` supports only geosite.dat, not geoip.dat
- `ext-ip` is supported to load geoip.dat
- `ip` is added to support matching of IPs
- `default` can be defined to match all

The value of map is the PAC return value.

For example, you can do:

```json
{
  "domain:example.com": "DIRECT",
  "ext:geosite.dat:category-ads": "PROXY 0.0.0.0:3421",
  "ext:geosite.dat:cn": "DIRECT",
  "ext:geosite.dat:gfw": "SOCKS5 127.0.0.1:1080",
  "ip:10.0.0.0/8": DIRECT,
  "ext-ip:geoip.dat:cn": "DIRECT",
  "default": "SOCKS5 127.0.0.1:1080"
}
```
