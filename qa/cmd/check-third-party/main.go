// Command check-third-party verifies that each third-party library has one owner.
//
// Enforces adr/technical/2026-10-25-third-party-libraries-have-one-owner.md.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

const (
	baselinePath = ".sombra/third-party-owners.txt"
	adrPath      = "adr/technical/2026-10-25-third-party-libraries-have-one-owner.md"
)

type owners map[string]map[string]bool

func main() {
	file := baselinePath
	if len(os.Args) > 1 {
		file = os.Args[1]
	}

	module := strings.TrimSpace(outputWithEnv([]string{"GOWORK=off"}, "go", "list", "-m"))
	if module == "" || strings.Contains(module, "\n") {
		die("cannot resolve a single Go module")
	}

	roots := moduleRoots()
	actual := collect(module, roots)
	baseline, err := loadBaseline(file)
	if err != nil {
		die(err.Error())
	}

	issues := compare(actual, baseline)
	if len(issues) == 0 {
		return
	}
	sort.Strings(issues)
	for _, issue := range issues {
		fmt.Println(issue)
	}
	fmt.Fprintf(os.Stderr, "check-third-party: %d issue(s); see %s\n", len(issues), adrPath)
	os.Exit(1)
}

func collect(module string, roots []string) owners {
	list := output("go", "list", "-f", "{{.ImportPath}}|{{join .Imports \",\"}}", "./...")
	result := make(owners)

	for _, line := range strings.Split(list, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 2)
		if len(parts) != 2 {
			continue
		}
		owner := ownerFor(parts[0], module)
		if owner == "" {
			continue
		}
		for _, imported := range strings.Split(parts[1], ",") {
			library, ok := libraryFor(imported, module, roots)
			if !ok {
				continue
			}
			addOwner(result, library, owner)
		}
	}
	return result
}

func moduleRoots() []string {
	list := output("go", "list", "-m", "-f", "{{.Path}}", "all")
	var roots []string
	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			roots = append(roots, line)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return len(roots[i]) > len(roots[j]) })
	return roots
}

func ownerFor(pkg, module string) string {
	prefix := module + "/"
	if !strings.HasPrefix(pkg, prefix) {
		return ""
	}
	rel := strings.TrimPrefix(pkg, prefix)

	switch {
	case strings.HasPrefix(rel, "cmd/"):
		return "cmd"
	case rel == "internal/runtime":
		return "runtime"
	case strings.HasPrefix(rel, "internal/frameworks/"):
		rest := strings.TrimPrefix(rel, "internal/frameworks/")
		unit := strings.Split(rest, "/")[0]
		if unit != "" {
			return unit
		}
	}
	return ""
}

func libraryFor(imported, module string, roots []string) (string, bool) {
	if imported == module || strings.HasPrefix(imported, module+"/") {
		return "", false
	}
	for _, root := range roots {
		if imported == root || strings.HasPrefix(imported, root+"/") {
			return root, true
		}
	}
	first := strings.Split(imported, "/")[0]
	if strings.Contains(first, ".") {
		return imported, true
	}
	return "", false
}

func addOwner(result owners, library, owner string) {
	if result[library] == nil {
		result[library] = make(map[string]bool)
	}
	result[library][owner] = true
}

func loadBaseline(file string) (owners, error) {
	result := make(owners)
	handle, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}
	defer func() { _ = handle.Close() }()

	scanner := bufio.NewScanner(handle)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		parts := strings.SplitN(text, "|", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s:%d: expected library|owner1,owner2", file, line)
		}
		library := strings.TrimSpace(parts[0])
		for _, owner := range strings.Split(parts[1], ",") {
			owner = strings.TrimSpace(owner)
			if library == "" || owner == "" {
				return nil, fmt.Errorf("%s:%d: empty library or owner", file, line)
			}
			addOwner(result, library, owner)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func compare(actual, baseline owners) []string {
	var issues []string

	for library, owners := range actual {
		if len(owners) <= 1 {
			continue
		}
		if baseline[library] == nil {
			issues = append(issues, fmt.Sprintf("%s is shared by %s; add a port and keep one owner", library, formatOwners(owners)))
			continue
		}
		if !sameOwners(owners, baseline[library]) {
			issues = append(issues, fmt.Sprintf("%s owners changed from %s to %s; update the baseline only when it shrinks", library, formatOwners(baseline[library]), formatOwners(owners)))
		}
	}

	for library, owners := range baseline {
		if len(actual[library]) <= 1 {
			issues = append(issues, fmt.Sprintf("baseline entry %s is no longer shared; remove it from %s", library, baselinePath))
			continue
		}
		if !sameOwners(actual[library], owners) {
			issues = append(issues, fmt.Sprintf("%s owners changed from %s to %s; update the baseline only when it shrinks", library, formatOwners(owners), formatOwners(actual[library])))
		}
	}

	return issues
}

func sameOwners(left, right map[string]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for owner := range left {
		if !right[owner] {
			return false
		}
	}
	return true
}

func formatOwners(owners map[string]bool) string {
	names := make([]string, 0, len(owners))
	for owner := range owners {
		names = append(names, owner)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

func output(name string, args ...string) string {
	return outputWithEnv(nil, name, args...)
}

func outputWithEnv(env []string, name string, args ...string) string {
	command := exec.Command(name, args...)
	command.Env = append(os.Environ(), env...)
	output, err := command.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
		os.Exit(1)
	}
	return string(output)
}

func die(message string) {
	fmt.Fprintln(os.Stderr, "check-third-party:", message)
	os.Exit(1)
}
