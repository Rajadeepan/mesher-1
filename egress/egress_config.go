/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package egress

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/go-chassis/go-chassis/core/lager"
	chassisTLS "github.com/go-chassis/go-chassis/core/tls"
	"github.com/go-chassis/go-chassis/pkg/istio/client"
	"github.com/go-chassis/go-chassis/pkg/util/iputil"
	"github.com/go-mesh/mesher/config"
	"github.com/go-mesh/mesher/config/model"
	"regexp"
	"strings"
	//"github.com/go-chassis/go-chassis/control/archaius"
	"github.com/go-chassis/go-chassis/control"
)

const (
	dns1123LabelMaxLength int    = 63
	dns1123LabelFmt       string = "[a-zA-Z0-9]([-a-z-A-Z0-9]*[a-zA-Z0-9])?"
	wildcardPrefix        string = "(\\*)?" + dns1123LabelFmt
	DefaultEgressType            = "cse"
	// EgressTLS defines tls prefix
	EgressTLS = "egress"
)

var (
	dns1123LabelRegexp   = regexp.MustCompile("^" + dns1123LabelFmt + "$")
	wildcardPrefixRegexp = regexp.MustCompile("^" + wildcardPrefix + "$")
)

// Init initialize Egress config
func Init() error {

	// init dests
	egressConfigFromFile := config.GetEgressConfig()
	BuildEgress(GetEgressType(egressConfigFromFile.Egress))

	if egressConfigFromFile != nil {
		if egressConfigFromFile.Destinations != nil {
			DefaultEgress.SetEgressRule(egressConfigFromFile.Destinations)

			var egressconfig []control.EgressConfig
			for _, v := range egressConfigFromFile.Destinations {
				var Ports []*control.EgressPort
				for _, v1 := range v {

					for _, v2 := range v1.Ports {
						p := control.EgressPort{
							Port:     (*v2).Port,
							Protocol: (*v2).Protocol,
						}
						Ports = append(Ports, &p)
					}
					c := control.EgressConfig{
						Hosts: v1.Hosts,
						Ports: Ports,
					}
					egressconfig = append(egressconfig, c)
				}

			}
			control.DefaultPanel.SaveToEgressCache(egressconfig)
		}
	}
	op, err := getSpecifiedOptions()
	if err != nil {
		return fmt.Errorf("Router options error: %v", err)
	}
	DefaultEgress.Init(op)
	// storing the egress rules based on host in two maps
	// one host having wild card and other without wildcard
	plainHosts, regexHosts = SplitEgressRules()
	lager.Logger.Info("Egress init success")
	return nil
}

//ValidateEgressRule validate the Egress rules of each service
func ValidateEgressRule(rules map[string][]*model.EgressRule) (bool, error) {
	for _, rule := range rules {
		for _, egressrule := range rule {
			if len(egressrule.Hosts) == 0 {
				return false, errors.New("Egress rule should have atleast one host")
			}
			for _, host := range egressrule.Hosts {
				err := ValidateHostName(host)
				if err != nil {
					return false, err
				}
			}
		}

	}
	return true, nil
}

//ValidateHostName validates the host
func ValidateHostName(host string) error {
	if len(host) > 255 {
		return fmt.Errorf("host name %q too long (max 255)", host)
	}
	if len(host) == 0 {
		return fmt.Errorf("empty host name not allowed")
	}

	parts := strings.SplitN(host, ".", 2)
	if !IsWildcardDNS1123Label(parts[0]) {
		return fmt.Errorf("host name %q invalid (label %q invalid)", host, parts[0])
	} else if len(parts) > 1 {
		err := validateDNS1123Labels(parts[1])
		return err
	}

	return nil

}

//IsWildcardDNS1123Label validate wild card label
func IsWildcardDNS1123Label(value string) bool {
	return len(value) <= dns1123LabelMaxLength && wildcardPrefixRegexp.MatchString(value)
}

//validateDNS1123Labels validate host
func validateDNS1123Labels(host string) error {
	for _, label := range strings.Split(host, ".") {
		if !IsDNS1123Label(label) {
			return fmt.Errorf("host name %q invalid (label %q invalid)", host, label)
		}
	}
	return nil
}

//IsDNS1123Label validate label
func IsDNS1123Label(value string) bool {
	return len(value) <= dns1123LabelMaxLength && dns1123LabelRegexp.MatchString(value)
}

// Options defines how to init Egress and its fetcher
type Options struct {
	Endpoints []string
	EnableSSL bool
	TLSConfig *tls.Config
	Version   string

	//TODO: need timeout for client
	// TimeOut time.Duration
}

// ToPilotOptions translate options to client options
func (o Options) ToPilotOptions() *client.PilotOptions {
	return &client.PilotOptions{Endpoints: o.Endpoints}
}

func getSpecifiedOptions() (opts Options, err error) {
	hosts, scheme, err := iputil.URIs2Hosts(strings.Split(config.GetEgressEndpoints(), ","))
	if err != nil {
		return
	}
	opts.Endpoints = hosts
	// TODO: envoy api v1 or v2
	// opts.Version = config.GetRouterAPIVersion()
	opts.TLSConfig, err = chassisTLS.GetTLSConfig(scheme, EgressTLS)
	if err != nil {
		return
	}
	if opts.TLSConfig != nil {
		opts.EnableSSL = true
	}
	return
}

// GetRouterType returns the type of router
func GetEgressType(egress model.Egress) string {
	if egress.Infra != "" {
		return egress.Infra
	}
	return DefaultEgressType
}
