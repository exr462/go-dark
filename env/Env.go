// Package env installs and runs a team's local environment: Rancher with
// its embedded k3s cluster in one docker container, set up like the real
// clusters behind the team's pipeline (NGINX Inc ingress with a mergeable
// master, a namespace, node label, Stakater Reloader) plus an Oracle Free
// database with the services' schema users.
//
// go-dark generates everything into an install directory the user can read
// and run without go-dark (docker compose, kubectl apply), never installs
// software itself, and never touches the hosts file silently.
package env

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/config"
)

// Env is a team environment with the names go-dark derives from it. Every
// container, compose project and kube context carries the team name, so two
// teams' environments (or the old sandbox) don't clash.
type Env struct {
	Team string
	config.Environment
	// DBPassword is the password of the database's SYS and schema users.
	DBPassword string
}

// Ports and names shared by every environment.
const (
	APIPort        = 6550 // k3s API on 127.0.0.1, the kube context's server
	HTTPNodePort   = 30080
	HTTPSNodePort  = 30443
	ContextPrefix  = "go-dark-" // kube contexts go-dark owns
	DefaultDBPass  = "changeit_local"
	OracleService  = "FREEPDB1"
	RancherImage   = "rancher/rancher:latest"
	OracleImage    = "gvenzl/oracle-free:23-slim"
	NginxChartVer  = "2.8.1"
	ReloaderVer    = "2.2.19"
	TLSSecret      = "api-tls"
	MasterIngress  = "api-master"
	K3sKubeconfig  = "/etc/rancher/k3s/k3s.yaml"
	ContainerdSock = "/run/k3s/containerd/containerd.sock"
)

// New builds the Env of a team environment.
func New(te config.TeamEnvironment, dbPassword string) Env {
	if dbPassword == "" {
		dbPassword = DefaultDBPass
	}
	return Env{Team: te.Team, Environment: te.Environment.WithDefaults(), DBPassword: dbPassword}
}

// Project is the docker compose project name.
func (e Env) Project() string { return ContextPrefix + e.Team }

// Container is the docker container running Rancher and k3s.
func (e Env) Container() string { return e.Project() + "-rancher" }

// Context is the kube context go-dark writes and deploys through.
func (e Env) Context() string { return ContextPrefix + e.Team }

// IngressClass is the environment's IngressClass, set as the default class
// because the charts only map the real environments to a class.
func (e Env) IngressClass() string { return "nginx-" + e.Name }

// NodeLabel is the node label the charts' nodeSelector wants.
func (e Env) NodeLabel() string { return "belgiantrain.be/env-" + e.Name + "=selected" }

// EnvironmentLabel labels the namespace and platform objects.
func (e Env) EnvironmentLabel() string { return "belgiantrain.be/environment=" + e.Name }

// APIHost is the services' shared API host.
func (e Env) APIHost() string { return e.Host("api") }

// RancherHost is the Rancher UI's host.
func (e Env) RancherHost() string { return e.Host("rancher") }

// HostsLine is the hosts-file line that points every host at this machine.
func (e Env) HostsLine() string { return "127.0.0.1 " + strings.Join(e.HostNames(), " ") }

// APIURL is where the services answer.
func (e Env) APIURL() string { return "https://" + e.APIHost() }

// DBURL is the JDBC URL of the environment's database as seen from the
// services' pods.
func (e Env) DBURL() string {
	return "jdbc:oracle:thin:@oracle." + e.Namespace + ".svc:1521/" + OracleService
}

// HasDatabase reports whether the environment runs a database.
func (e Env) HasDatabase() bool { return e.Database.Type != config.DatabaseNone }

// Root is where all environments are installed: ~/.local/share/go-dark/env.
func Root() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "go-dark", "env"), nil
}

// InstallDir is the environment's install directory, one per team.
func (e Env) InstallDir() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, e.Team), nil
}

// Paths inside the install directory.
const (
	ComposeFile   = "docker-compose.yaml"
	K3sConfigFile = "k3s/config.yaml"
	CertsDir      = "certs"
	CertFile      = "certs/tls.crt"
	KeyFile       = "certs/tls.key"
	PlatformDir   = "platform"
	ChartsDir     = "charts"
	ReadmeFile    = "README.md"
)
