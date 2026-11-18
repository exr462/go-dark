package maven

import (
	"os/exec"
	"strconv"
	"strings"
)

// LatestReleaseTag returns the highest release version among the repo's git
// tags named "<artifactId>-X.Y.Z", "vX.Y.Z" or "X.Y.Z" (pre-releases such as
// SNAPSHOT or ALPHA are ignored), or "" if there is none.
func LatestReleaseTag(repoDir, artifactID string) string {
	out, err := exec.Command("git", "-C", repoDir, "tag", "--list").Output()
	if err != nil {
		return ""
	}
	best := ""
	for _, tag := range strings.Fields(string(out)) {
		v := strings.TrimPrefix(tag, artifactID+"-")
		v = strings.TrimPrefix(v, "v")
		if isRelease(v) && (best == "" || CompareVersions(v, best) > 0) {
			best = v
		}
	}
	return best
}

func isRelease(v string) bool {
	if v == "" {
		return false
	}
	for _, part := range strings.Split(v, ".") {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

// CompareVersions compares dotted versions numerically where possible
// ("3.10.0" > "3.9.1"); a pre-release sorts below its release
// ("3.6.0-SNAPSHOT" < "3.6.0").
func CompareVersions(a, b string) int {
	aBase, aPre, _ := strings.Cut(a, "-")
	bBase, bPre, _ := strings.Cut(b, "-")
	ap, bp := strings.Split(aBase, "."), strings.Split(bBase, ".")
	for i := range max(len(ap), len(bp)) {
		var x, y string
		if i < len(ap) {
			x = ap[i]
		}
		if i < len(bp) {
			y = bp[i]
		}
		xn, xerr := strconv.Atoi(x)
		yn, yerr := strconv.Atoi(y)
		switch {
		case xerr == nil && yerr == nil && xn != yn:
			return cmp(xn, yn)
		case (xerr != nil || yerr != nil) && x != y:
			return strings.Compare(x, y)
		}
	}
	switch {
	case aPre == bPre:
		return 0
	case aPre == "":
		return 1
	case bPre == "":
		return -1
	}
	return strings.Compare(aPre, bPre)
}

func cmp(a, b int) int {
	if a < b {
		return -1
	}
	return 1
}
