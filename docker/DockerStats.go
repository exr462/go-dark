package docker

type DockerStats struct {
	CPU      string // e.g. "12.4%"
	Memory   string // e.g. "1.45GB / 16GB"
	Running  int    // Number of active running containers
	Services int    // Number of total configured containers
}
