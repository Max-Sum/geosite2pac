package main

import (
	"io"
	"text/template"
)

var pacTemplate = `var geoip = {{ printf "%s" .GeoIP }}
var geosite = {{ printf "%s" .GeoSite }}
var rules = {{ printf "%s" .Rules }}

// Domain matching
function MatchHost(host, rule) {
    var idxColon = rule.indexOf(':');
    var method = rule.substring(0, idxColon);
    var rule = rule.substring(idxColon + 1);
    switch (method) {
        case "":
            return host.includes(rule);
        case "full":
            return rule === host;
        case "domain":
            return rule === host || dnsDomainIs(host, '.' + rule);
        case "regexp":
            return (new RegExp(rule)).test(host);
        default:
            return false;
    }
}

function MatchHostByRules(host, rules) {
    var match = function (rule) { return MatchHost(host, rule); }
    return rules.some(match);
}

// IP-address matching
function convert_addr(ipchars) {
    var bytes = ipchars.split('.');
    return ((bytes[0] & 0xff) << 24) |
        ((bytes[1] & 0xff) << 16) |
        ((bytes[2] & 0xff) << 8) |
        (bytes[3] & 0xff);
}

function MatchIPv4(ipaddr, cidr) {
    var idxSlash = cidr.indexOf('/');
    var baseip = cidr;
    var ignoreBits = 0;
    if (idxSlash > 0) {
        baseip = cidr.substring(0, idxSlash);
        ignoreBits = 32 - parseInt(cidr.substring(idxSlash));
    }
    return (convert_addr(ipaddr) >> ignoreBits) === (convert_addr(baseip) >> ignoreBits);
}

function MatchIPv4ByRules(ipaddr, cidrs) {
    var match = function (cidr) { return MatchIPv4(ipaddr, cidr) };
    return cidrs.some(match);
}

function FindProxyForURL(url, host) {
    var ipaddr = null;
    for (var rule in rules) {
        var idxColon = rule.indexOf(':');
        var method = rule.substring(0, idxColon);
        var name = rule.substring(idxColon + 1);
        var reverse = false;
        if (name.substring(0, 1) === "!") {
            name = name.substring(1);
            reverse = true;
        }
        var matched = false;
        switch (method) {
            case "geosite":
                var matched = MatchHostByRules(host, geosite[name]);
                break;
            case "geoip":
                if (ipaddr === null) {
                    if (!isResolvable(host)) { break; }
                    ipaddr = dnsResolve(host);
                }
                matched = MatchIPv4ByRules(ipaddr, geoip[name]);
                break;
            case "ip":
                if (ipaddr === null) {
                    if (!isResolvable(host)) { break; }
                    ipaddr = dnsResolve(host);
                }
                matched = MatchIPv4(ipaddr, name);
                break;
            default:
                if (rule === "default") {
                    return rules[rule];
                }
                matched = MatchHost(host, name);
                break;
        }
        if ((reverse && (!matched)) || ((!reverse) && matched)) {
            return rules[rule];
        }
    }
    return "DIRECT";
}

// Production steps of ECMA-262, Edition 5, 15.4.4.17
// Reference: http://es5.github.io/#x15.4.4.17
if (!Array.prototype.some) {
    Array.prototype.some = function (fun/*, thisArg*/) {
        'use strict';

        if (this == null) {
            throw new TypeError('Array.prototype.some called on null or undefined');
        }

        if (typeof fun !== 'function') {
            throw new TypeError();
        }

        var t = Object(this);
        var len = t.length >>> 0;

        var thisArg = arguments.length >= 2 ? arguments[1] : void 0;
        for (var i = 0; i < len; i++) {
            if (i in t && fun.call(thisArg, t[i], i, t)) {
                return true;
            }
        }

        return false;
    };
}`

type tmplValue struct {
	GeoSite []byte
	GeoIP   []byte
	Rules   []byte
}

func output(v *tmplValue, wr io.Writer) error {
	tmpl, err := template.New("pac").Parse(pacTemplate)
	if err != nil {
		return err
	}
	err = tmpl.Execute(wr, v)
	if err != nil {
		return err
	}
	return nil
}
