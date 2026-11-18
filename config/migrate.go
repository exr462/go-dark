package config

// legacyProject mirrors the registry that used to be hardcoded in go-dark
// before team manifests existed. It is only used to migrate old configs.
type legacyProject struct {
	Name         string
	Deployable   bool
	Buildable    bool
	Dependencies []string
}

var legacyRegistry = []legacyProject{
	{"ypto-maven-parent", false, true, nil},
	{"contract-first-api-parent", false, true, []string{"ypto-maven-parent"}},
	{"til-purchase", true, true, []string{"contract-first-api-parent"}},
	{"til-product", true, true, []string{"contract-first-api-parent"}},
	{"til-smm", true, true, []string{"contract-first-api-parent"}},
	{"til-fdm", true, true, []string{"contract-first-api-parent"}},
	{"til-b2b-traveler", true, true, []string{"contract-first-api-parent"}},
	{"til-stop-place", true, true, []string{"contract-first-api-parent"}},
	{"til-channel", true, true, []string{"contract-first-api-parent"}},
	{"til-b2c-voucher", true, true, []string{"contract-first-api-parent"}},
	{"til-fulfillment", true, true, []string{"contract-first-api-parent"}},
	{"til-document", true, true, []string{"contract-first-api-parent"}},
	{"til-dtm", true, true, []string{"contract-first-api-parent"}},
	{"til-notification", true, true, []string{"contract-first-api-parent"}},
	{"til-order", true, true, []string{"contract-first-api-parent"}},
	{"til-b2b-voucher", true, true, []string{"contract-first-api-parent"}},
	{"til-authorization", true, true, []string{"contract-first-api-parent"}},
	{"til-mms", true, true, []string{"contract-first-api-parent"}},
	{"til-report", true, true, []string{"contract-first-api-parent"}},
	{"til-travel-time", true, true, []string{"contract-first-api-parent"}},
	{"til-payment", true, true, []string{"contract-first-api-parent"}},
	{"til-global-aks", true, false, []string{"contract-first-api-parent"}},
	{"til-communication", true, true, []string{"contract-first-api-parent"}},
	{"til-cash-device", true, true, []string{"contract-first-api-parent"}},
	{"til-voucher", true, true, []string{"contract-first-api-parent"}},
}

// migrateLegacyProjects upgrades projects saved before IDs existed into
// local projects, restoring the dependency metadata the old hardcoded
// registry provided. It reports whether anything changed.
func migrateLegacyProjects(projects []Project) bool {
	legacy := make(map[string]legacyProject, len(legacyRegistry))
	for _, lp := range legacyRegistry {
		legacy[lp.Name] = lp
	}

	changed := false
	for i := range projects {
		p := &projects[i]
		if p.ID != "" {
			continue
		}
		changed = true
		p.Source = LocalSource
		p.ID = ProjectID(LocalSource, p.Name)
		if p.Type == "java" || p.Type == "" {
			p.Type = "maven"
		}
		if lp, ok := legacy[p.Name]; ok {
			p.Buildable = lp.Buildable
			p.Deployable = lp.Deployable
			p.DependsOn = nil
			for _, d := range lp.Dependencies {
				p.DependsOn = append(p.DependsOn, ProjectID(LocalSource, d))
			}
		} else {
			p.Buildable = true
		}
	}
	return changed
}
