package component

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/env"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/color"
	"github.com/exr462/go-dark/ui/decorator"
)

// RenderEnvModal is the installer screen of the team's local environment:
// where it is installed, the preflight, what runs, and the lifecycle keys.
func RenderEnvModal(ui *model.UI) string {
	width := max(ui.WindowWidth-8, 20)
	var b strings.Builder
	title := fmt.Sprintf("🏗️ Local environment (%s)", action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenEnv))
	e, ok := ui.CurrentEnv()
	if !ok {
		b.WriteString(decorator.Title.Render(title) + "\n")
		b.WriteString(decorator.Description.Render("No subscribed team manifest declares an \"environment\" section, so there is nothing to install. See the README (Teams & projects) for the schema.") + "\n\n")
		b.WriteString(sessHintStyle.Render(fmt.Sprintf("[%s] Back", sessKeyHint.Render(action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)))))
		return envBox(ui, b.String())
	}

	b.WriteString(decorator.Title.Render(fmt.Sprintf("%s  %q of team %s", title, e.Name, e.Team)))
	if n := len(ui.Config.Environments); n > 1 {
		b.WriteString(decorator.Meta.Render(fmt.Sprintf("   (%d/%d, tab switches)", ui.EnvIndex+1, n)))
	}
	b.WriteString("\n")
	b.WriteString(decorator.Description.Render(fmt.Sprintf(
		"Rancher + k3s in docker container %s, set up like %s: NGINX Inc ingress (%s), namespace %s, Reloader%s. The files are generated into the install dir and can be run without go-dark.",
		e.Container(), e.ConfigFrom, e.IngressClass(), e.Namespace, map[bool]string{true: ", Oracle Free", false: ""}[e.HasDatabase()])) + "\n")
	cloned, total := ui.EnvProjects(e.Team)
	b.WriteString(fmt.Sprintf("  %s %s\n", decorator.Bold.Render("Install dir "), model.EnvDir(e)))
	b.WriteString(fmt.Sprintf("  %s %s/<service>   https://%s   kubectl --context %s -n %s\n", decorator.Bold.Render("URLs        "), e.APIURL(), e.RancherHost(), e.Context(), e.Namespace))
	b.WriteString(fmt.Sprintf("  %s %d of %d deployable services cloned (their config/%s gives the database users)\n\n", decorator.Bold.Render("Services    "), cloned, total, e.ConfigFrom))

	switch {
	case !ui.EnvChecked && ui.EnvChecking:
		b.WriteString(decorator.Meta.Render("⏳ Checking docker, tools, ports and the environment...") + "\n")
	case !ui.EnvChecked:
		b.WriteString(decorator.Meta.Render("  (not checked yet - r)") + "\n")
	default:
		b.WriteString(decorator.Section.Render("Preflight") + "\n")
		for _, c := range ui.EnvChecks {
			line := fmt.Sprintf("  %s %-14s %s", checkBadge(c.State), c.Name, c.Detail)
			if c.Fix != "" {
				line += decorator.Meta.Render("  → " + c.Fix)
			}
			b.WriteString(ansi.Truncate(line, width, "…") + "\n")
		}
		if len(ui.EnvConflicts) > 0 {
			b.WriteString(ansi.Truncate(decorator.Warning.Render("  ⚠ Another compose project holds the ports (the old sandbox?) - o stops it; its volumes and files are kept"), width, "…") + "\n")
		}
		b.WriteString("\n" + decorator.Section.Render("Status") + "\n")
		b.WriteString(envStatusLines(ui, e, width))
	}

	b.WriteString("\n")
	if ui.EnvActionID != 0 && ui.Shells != nil {
		if s := ui.Shells.Get(ui.EnvActionID); s != nil {
			state := decorator.Success.Render("✅ finished")
			switch {
			case s.Running():
				state = decorator.Warning.Render("⏳ running")
			case s.ExitCode() != 0:
				state = decorator.Error.Render(fmt.Sprintf("❌ failed (exit %d)", s.ExitCode()))
			}
			b.WriteString(fmt.Sprintf("  %s %s %s %s\n", decorator.Bold.Render("Last action "), ui.EnvAction, state, decorator.Meta.Render(fmt.Sprintf("session #%d - l opens the log", s.ID))))
		}
	}
	if ui.EnvHint != "" {
		b.WriteString(ansi.Truncate(decorator.Warning.Render("  Run yourself: "+ui.EnvHint), width, "…") + "\n")
	}
	if ui.EnvConfirm != "" {
		b.WriteString(decorator.Warning.Render("  "+ui.StatusMsg) + "\n")
	}

	esc := action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape)
	hints := []string{
		fmt.Sprintf("[%s] Install / repair", sessKeyHint.Render("i")),
		fmt.Sprintf("[%s] Start", sessKeyHint.Render("s")),
		fmt.Sprintf("[%s] Stop", sessKeyHint.Render("t")),
		fmt.Sprintf("[%s] Uninstall", sessKeyHint.Render("u")),
		fmt.Sprintf("[%s] Hosts file", sessKeyHint.Render("h")),
	}
	if len(ui.EnvConflicts) > 0 {
		hints = append(hints, fmt.Sprintf("[%s] Stop the old sandbox", sessKeyHint.Render("o")))
	}
	if ui.EnvActionID != 0 {
		hints = append(hints, fmt.Sprintf("[%s] Log", sessKeyHint.Render("l")))
	}
	hints = append(hints, fmt.Sprintf("[%s] Re-check", sessKeyHint.Render("r")), fmt.Sprintf("[%s] Back", sessKeyHint.Render(esc)))
	b.WriteString(sessHintStyle.Render(strings.Join(hints, " | ")))
	return envBox(ui, b.String())
}

func envStatusLines(ui *model.UI, e env.Env, width int) string {
	st := ui.EnvStatus
	var b strings.Builder
	installed := decorator.Meta.Render("not installed (i installs it)")
	if st.Installed {
		installed = decorator.Success.Render("installed")
	}
	container := decorator.Meta.Render(st.Container)
	switch st.Container {
	case "running":
		container = decorator.Success.Render("running")
	case "stopped":
		container = decorator.Warning.Render("stopped (s starts it)")
	}
	b.WriteString(fmt.Sprintf("  %s %s   %s %s\n", decorator.Bold.Render("Files      "), installed, decorator.Bold.Render("Container"), container))
	if st.Container == "running" {
		k3s := decorator.Warning.Render("booting (r re-checks)")
		if st.K3sReady {
			k3s = decorator.Success.Render("ready")
		}
		line := fmt.Sprintf("  %s %s", decorator.Bold.Render("k3s        "), k3s)
		if st.EtcdBytes > 0 {
			line += decorator.Meta.Render("   etcd " + env.HumanBytes(st.EtcdBytes))
		}
		b.WriteString(line + "\n")
		for _, kind := range st.Pressure {
			b.WriteString(ansi.Truncate(decorator.Error.Render(fmt.Sprintf("  ⚠ Node %s, pods are evicted: %s", kind, env.PressureHint(kind))), width, "…") + "\n")
		}
		if st.K3sReady {
			b.WriteString(fmt.Sprintf("  %s %s   %s %s\n", decorator.Bold.Render("Ingress    "), readyStyle(st.Ingress), decorator.Bold.Render("Database"), readyStyle(st.Database)))
			releases := "none"
			if len(st.Releases) > 0 {
				releases = strings.Join(st.Releases, ", ")
			}
			b.WriteString(ansi.Truncate(fmt.Sprintf("  %s %s", decorator.Bold.Render("Releases   "), releases), width, "…") + "\n")
		}
	}
	if len(st.Missing) > 0 {
		b.WriteString(ansi.Truncate(decorator.Warning.Render(fmt.Sprintf("  Hosts file  missing %s - h adds \"%s\"", strings.Join(st.Missing, ", "), e.HostsLine())), width, "…") + "\n")
	} else {
		b.WriteString(fmt.Sprintf("  %s %s\n", decorator.Bold.Render("Hosts file "), decorator.Success.Render("every host points here")))
	}
	return b.String()
}

func readyStyle(s string) string {
	switch {
	case s == "ready" || s == "none":
		return decorator.Success.Render(s)
	case strings.HasPrefix(s, "not ready"):
		return decorator.Warning.Render(s)
	}
	return decorator.Meta.Render(s)
}

func checkBadge(s env.CheckState) string {
	switch s {
	case env.CheckWarn:
		return decorator.Warning.Render("⚠")
	case env.CheckFail:
		return decorator.Error.Render("❌")
	}
	return decorator.Success.Render("✅")
}

func envBox(ui *model.UI, body string) string {
	return lipgloss.Place(
		ui.WindowWidth, ui.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		decorator.ModalBox.Width(ui.WindowWidth-4+decorator.BorderFrame).Render(body),
		decorator.WhiteSpace,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(color.Crust)),
	)
}
