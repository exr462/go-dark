package config

type Project struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	GitURL     string `json:"git_url"`
	JDKName    string `json:"jdk_name"`
	MavenName  string `json:"maven_name"`
	Deployable bool   `json:"deployable"`
	Fetched    bool   `json:"fetched"`
}
