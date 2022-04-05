package main

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path"
	"path/filepath"
	"strings"

	"github.com/iancoleman/orderedmap"
	router "github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
)

type KeyVal struct {
	Key string
	Val interface{}
}

// Define an ordered map
type OrderedMap []KeyVal

func GeoSite2Str(g *router.GeoSite) []string {
	strarr := make([]string, 0, len(g.Domain))
	for _, rule := range g.Domain {
		ruleVal := strings.TrimSpace(rule.GetValue())
		if len(ruleVal) == 0 {
			continue
		}
		switch rule.Type {
		case router.Domain_Plain:
			strarr = append(strarr, ruleVal)
		case router.Domain_Full:
			strarr = append(strarr, "full:"+ruleVal)
		case router.Domain_RootDomain:
			strarr = append(strarr, "domain:"+ruleVal)
		case router.Domain_Regex:
			strarr = append(strarr, "regexp:"+ruleVal)
		}
	}
	return strarr
}

func GeoIP2Str(g *router.GeoIP) []string {
	strarr := make([]string, 0, len(g.Cidr))
	for _, rule := range g.Cidr {
		ruleVal := fmt.Sprintf("%s/%d", net.IP(rule.GetIp()), rule.GetPrefix())
		if len(ruleVal) == 0 {
			continue
		}
		strarr = append(strarr, ruleVal)
	}
	return strarr
}

func convert(rulePath string, wr io.Writer) error {
	// Read rules
	rulesText, err := ioutil.ReadFile(rulePath)
	if err != nil {
		return err
	}
	var rules orderedmap.OrderedMap
	json.Unmarshal(rulesText, &rules)

	// File Path
	basepath, err := filepath.Abs(filepath.Dir(rulePath))
	if err != nil {
		return err
	}

	geoSiteLists := make(map[string]*router.GeoSiteList)
	geoIPLists := make(map[string]*router.GeoIPList)
	geoSites := make(map[string][]string)
	geoIPs := make(map[string][]string)
	newRules := orderedmap.New()

	for _, rule := range rules.Keys() {
		action, _ := rules.Get(rule)
		method := ""
		if strings.Contains(rule, ":") {
			s := strings.SplitN(rule, ":", 2)
			method, rule = s[0], s[1]
		}
		switch method {
		case "ext":
			s := strings.SplitN(rule, ":", 2)
			fn, tag := s[0], s[1]
			if _, ok := geoSiteLists[fn]; !ok {
				f, err := ioutil.ReadFile(path.Join(basepath, fn))
				if err != nil {
					return err
				}
				geoSiteLists[fn] = new(router.GeoSiteList)
				if err := proto.Unmarshal(f, geoSiteLists[fn]); err != nil {
					return err
				}
			}
			hash := sha1.Sum([]byte(fn + tag))
			hashstr := fmt.Sprintf("%.4x", hash)
			if _, ok := geoSites[hashstr]; !ok {
				var geosite *router.GeoSite
				for _, g := range geoSiteLists[fn].Entry {
					if g.GetCountryCode() == strings.ToUpper(tag) {
						geosite = g
						break
					}
				}
				if geosite == nil {
					return errors.New("tag:" + tag + " is not found in " + fn)
				}
				geoSites[hashstr] = GeoSite2Str(geosite)
			}
			newRules.Set("geosite:"+hashstr, action)
		case "ext-ip":
			s := strings.SplitN(rule, ":", 2)
			fn, tag := s[0], s[1]
			if _, ok := geoIPLists[fn]; !ok {
				f, err := ioutil.ReadFile(path.Join(basepath, fn))
				if err != nil {
					return err
				}
				geoIPLists[fn] = new(router.GeoIPList)
				if err := proto.Unmarshal(f, geoIPLists[fn]); err != nil {
					return err
				}
			}
			hash := sha1.Sum([]byte(fn + tag))
			hashstr := fmt.Sprintf("%.4x", hash)
			if _, ok := geoIPLists[hashstr]; !ok {
				var geoip *router.GeoIP
				for _, g := range geoIPLists[fn].Entry {
					if g.GetCountryCode() == strings.ToUpper(tag) {
						geoip = g
						break
					}
				}
				if geoip == nil {
					return errors.New("tag:" + tag + " is not found in " + fn)
				}
				geoIPs[hashstr] = GeoIP2Str(geoip)
			}
			newRules.Set("geoip:"+hashstr, action)
		case "":
			newRules.Set(rule, action)
		default:
			newRules.Set(method+":"+rule, action)
		}
	}
	// Output
	v := &tmplValue{}
	v.GeoSite, err = json.Marshal(geoSites)
	if err != nil {
		return err
	}
	v.GeoIP, err = json.Marshal(geoIPs)
	if err != nil {
		return err
	}
	v.Rules, err = json.Marshal(newRules)
	if err != nil {
		return err
	}
	return output(v, wr)
}
