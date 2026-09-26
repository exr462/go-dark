package config

type AvailableProject struct {
	Name         string
	Deployable   bool
	Buildable    bool
	Dependencies []string
}
