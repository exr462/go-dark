package config

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Environment is the team's local environment: a Rancher/k3s cluster set up
// like the real ones behind the team's pipeline, declared once in the team
// manifest ("environment") so every member installs the same thing with
// go-dark. Nothing in it is specific to one team: hosts and names come from
// the manifest, with defaults derived from Name and Application.
type Environment struct {
	Name        string `json:"name"`        // chart environment, e.g. lcl (ypto.env, node label, ingress class suffix)
	Application string `json:"application"` // the application the services belong to, e.g. til (umbrella release name)
	ConfigFrom  string `json:"config_from"` // real environment whose values, config and secrets the charts are copied from, e.g. dev
	Domain      string `json:"domain"`      // DNS domain of the environment, e.g. lcl-intern-belgiantrain.be
	// Hosts are the hostnames the environment answers on, by role. "api" is
	// the services' shared API host (the charts' master ingress), "rancher"
	// the Rancher UI. {domain}, {application} and {name} are expanded.
	Hosts     map[string]string `json:"hosts"`
	Namespace string            `json:"namespace"` // where the services are deployed, e.g. til-lcl
	// ValuesFiles maps a real environment to the chart values file its
	// pipeline deploys with, e.g. dev -> values.Development.yaml.
	ValuesFiles map[string]string   `json:"values_files"`
	Ingress     EnvironmentIngress  `json:"ingress"`
	Database    EnvironmentDatabase `json:"database"`
	// DNSUpstream is the resolver CoreDNS forwards to. Empty auto-detects
	// Docker's resolver from the cluster container (stock k3s forwarding
	// fails on hosts where public DNS is blocked).
	DNSUpstream string `json:"dns_upstream,omitempty"`
}

// EnvironmentIngress describes the ingress controller the charts expect.
type EnvironmentIngress struct {
	// Controller is the controller family; only "nginx-inc" (NGINX Inc,
	// nginx.org/* annotations, mergeable ingresses) is supported.
	Controller string `json:"controller"`
	// StripPrefix removes the first path segment (/<service>) before
	// proxying and sends it as X-Forwarded-Prefix, for charts that route
	// /<service> with nginx.org/rewrites.
	StripPrefix bool `json:"strip_prefix"`
}

// EnvironmentDatabase describes the environment's database.
type EnvironmentDatabase struct {
	// Type is "oracle-free" (gvenzl/oracle-free in the namespace) or
	// "none".
	Type string `json:"type"`
	// UsersFrom is where the schema users come from: "config" reads each
	// service's spring.datasource.username / spring.liquibase.user from its
	// helm/*/config/<config_from>/*.properties.
	UsersFrom string `json:"users_from"`
	// Users are extra users (app user and schema owner) to create as well.
	Users []DatabaseUser `json:"users,omitempty"`
}

// DatabaseUser is an application user and the schema owner it works in
// (the same user when the service owns its schema).
type DatabaseUser struct {
	App   string `json:"app"`
	Owner string `json:"owner,omitempty"`
}

// Supported controller and database types.
const (
	IngressNginxInc    = "nginx-inc"
	DatabaseOracleFree = "oracle-free"
	DatabaseNone       = "none"
	UsersFromConfig    = "config"
)

// DefaultValuesFiles is the pipeline's values file per environment.
var DefaultValuesFiles = map[string]string{
	"dev": "values.Development.yaml",
	"tst": "values.Test.yaml",
	"acc": "values.Acceptance.yaml",
	"prd": "values.Production.yaml",
}

// WithDefaults fills in what the manifest left out: name lcl, config from
// dev, namespace <application>-<name>, the api and rancher hosts, the
// pipeline's values files, the NGINX Inc controller and an Oracle Free
// database with users from the services' config.
func (e Environment) WithDefaults() Environment {
	set := func(v *string, def string) {
		if *v == "" {
			*v = def
		}
	}
	set(&e.Name, "lcl")
	set(&e.ConfigFrom, "dev")
	set(&e.Namespace, e.Application+"-"+e.Name)
	hosts := map[string]string{"api": "api.{application}.{domain}", "rancher": "rancher.{domain}"}
	for k, v := range e.Hosts {
		hosts[k] = v
	}
	e.Hosts = hosts
	if len(e.ValuesFiles) == 0 {
		e.ValuesFiles = DefaultValuesFiles
	}
	set(&e.Ingress.Controller, IngressNginxInc)
	set(&e.Database.Type, DatabaseOracleFree)
	set(&e.Database.UsersFrom, UsersFromConfig)
	return e
}

// Host returns the hostname for a role (api, rancher, ...) with the
// placeholders expanded, or "" for an unknown role.
func (e Environment) Host(role string) string {
	h, ok := e.Hosts[role]
	if !ok {
		return ""
	}
	return strings.NewReplacer("{domain}", e.Domain, "{application}", e.Application, "{name}", e.Name).Replace(h)
}

// HostNames lists every hostname, sorted and without duplicates.
func (e Environment) HostNames() []string {
	var names []string
	for role := range e.Hosts {
		if h := e.Host(role); h != "" && !slices.Contains(names, h) {
			names = append(names, h)
		}
	}
	slices.Sort(names)
	return names
}

// ValuesFile is the chart values file of the environment the charts are
// configured from.
func (e Environment) ValuesFile() string { return e.ValuesFiles[e.ConfigFrom] }

var dnsLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
var hostName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

// Validate checks an environment after WithDefaults: names must be DNS
// labels (they become namespaces, labels and container names), hosts valid
// hostnames, and the controller and database types supported.
func (e Environment) Validate() error {
	for what, v := range map[string]string{"name": e.Name, "application": e.Application, "namespace": e.Namespace, "config_from": e.ConfigFrom} {
		if !dnsLabel.MatchString(v) {
			return fmt.Errorf("environment %s %q must be a lowercase DNS label", what, v)
		}
	}
	if !hostName.MatchString(e.Domain) {
		return fmt.Errorf("environment domain %q is not a valid domain name", e.Domain)
	}
	for _, role := range []string{"api", "rancher"} {
		if e.Host(role) == "" {
			return fmt.Errorf("environment hosts must include %q", role)
		}
	}
	for role := range e.Hosts {
		if h := e.Host(role); !hostName.MatchString(h) {
			return fmt.Errorf("environment host %s %q is not a valid hostname", role, h)
		}
	}
	if e.ValuesFile() == "" {
		return fmt.Errorf("environment values_files has no entry for config_from %q", e.ConfigFrom)
	}
	if e.Ingress.Controller != IngressNginxInc {
		return fmt.Errorf("environment ingress controller %q is not supported (only %s)", e.Ingress.Controller, IngressNginxInc)
	}
	if t := e.Database.Type; t != DatabaseOracleFree && t != DatabaseNone {
		return fmt.Errorf("environment database type %q is not supported (%s or %s)", t, DatabaseOracleFree, DatabaseNone)
	}
	if e.Database.UsersFrom != UsersFromConfig {
		return fmt.Errorf("environment database users_from %q is not supported (only %s)", e.Database.UsersFrom, UsersFromConfig)
	}
	for _, u := range e.Database.Users {
		if !dbUser.MatchString(u.App) || (u.Owner != "" && !dbUser.MatchString(u.Owner)) {
			return fmt.Errorf("environment database user %+v: names must be identifiers", u)
		}
	}
	return nil
}

var dbUser = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_$#]{0,29}$`)

// TeamEnvironment is a team's environment, with the team it belongs to.
type TeamEnvironment struct {
	Team string
	Environment
}

// Environments returns the environment of every subscribed team whose
// manifest declares one, in subscription order.
func Environments(manifests []Manifest) []TeamEnvironment {
	var envs []TeamEnvironment
	for _, m := range manifests {
		if m.Environment != nil {
			envs = append(envs, TeamEnvironment{Team: m.Team, Environment: *m.Environment})
		}
	}
	return envs
}
