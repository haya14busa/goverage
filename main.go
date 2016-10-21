package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/tools/cover"
)

const usageMessage = "" +
	`Usage:	goverage [flags] -coverprofile=coverage.out package
`

var (
	coverprofile string
	covermode    string
	cpu          string
	parallel     string
	timeout      string
	short        bool
	v            bool
)

func init() {
	flag.StringVar(&coverprofile, "coverprofile", "", "Write a coverage profile to the file after all tests have passed. (required)")
	flag.StringVar(&covermode, "covermode", "", "sent as covermode argument to go test")
	flag.StringVar(&cpu, "cpu", "", "sent as cpu argument to go test")
	flag.StringVar(&parallel, "parallel", "", "sent as parallel argument to go test")
	flag.StringVar(&timeout, "timeout", "", "sent as timeout argument to go test")
	flag.BoolVar(&short, "short", false, "sent as short argument to go test")
	flag.BoolVar(&v, "v", false, "sent as v argument to go test")
}

func usage() {
	fmt.Fprintln(os.Stderr, usageMessage)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if err := run(coverprofile, flag.Arg(0), covermode, cpu, parallel, timeout, short, v); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(coverprofile, pkg, covermode, cpu, parallel, timeout string, short, v bool) error {
	if coverprofile == "" {
		usage()
		return nil
	}
	file, err := os.Create(coverprofile)
	if err != nil {
		return err
	}
	defer file.Close()
	// pkgs is packages to run tests and get coverage.
	pkgs, err := getPkgs(pkg)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return nil
	}
	coverpkg := strings.Join(pkgs, ",")
	cpss := make([][]*cover.Profile, len(pkgs))
	for i, pkg := range pkgs {
		cps, err := coverage(coverpkg, pkg, covermode, cpu, parallel, timeout, short, v)
		if err == nil {
			cpss[i] = cps
		}
	}
	dumpcp(file, mergeProfiles(cpss))
	return nil
}

// getPkgs returns packages for mesuring coverage. Returned packages doesn't
// contain vendor packages.
func getPkgs(pkg string) ([]string, error) {
	if pkg == "" {
		pkg = "./..."
	}
	out, err := exec.Command("go", "list", pkg).CombinedOutput()
	if err != nil {
		return nil, err
	}
	allPkgs := strings.Split(strings.Trim(string(out), "\n"), "\n")
	pkgs := make([]string, 0, len(allPkgs))
	for _, p := range allPkgs {
		if !strings.Contains(p, "/vendor/") {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, nil
}

// coverage runs test for the given pkg and returns cover profile.
func coverage(coverpkg, pkg, covermode, cpu, parallel, timeout string, short, v bool) ([]*cover.Profile, error) {
	f, err := ioutil.TempFile("", "goverage")
	if err != nil {
		return nil, err
	}
	f.Close()
	defer os.Remove(f.Name())
	args := []string{
		"test", pkg,
		"-coverpkg", coverpkg,
		"-coverprofile", f.Name(),
	}
	if covermode != "" {
		args = append(args, "-covermode", covermode)
	}
	if cpu != "" {
		args = append(args, "-cpu", cpu)
	}
	if parallel != "" {
		args = append(args, "-parallel", parallel)
	}
	if timeout != "" {
		args = append(args, "-timeout", timeout)
	}
	if short {
		args = append(args, "-short")
	}
	if v {
		args = append(args, "-v")
	}
	cmd := exec.Command("go", args...)
	if v {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return cover.ParseProfiles(f.Name())
}

// mergeProfiles merges cover profiles. It assumes target packages of each
// cover profile are same and sorted.
func mergeProfiles(cpss [][]*cover.Profile) []*cover.Profile {
	// skip head empty profiles ([no test files])
	for i, cps := range cpss {
		if len(cps) == 0 {
			continue // [no test files]
		}
		cpss = cpss[i:]
		break
	}
	if len(cpss) == 0 {
		return nil // empty
	} else if len(cpss) == 1 {
		return cpss[0] // only one profile
	}
	result, rest := cpss[0], cpss[1:]
	for i, profile := range result {
		for _, cps := range rest {
			if len(cps) == 0 {
				continue // [no test files]
			}
			cp := cps[i]
			for j, block := range cp.Blocks {
				switch profile.Mode {
				case "set":
					profile.Blocks[j].Count |= block.Count
				case "count", "atomic":
					profile.Blocks[j].Count += block.Count
				}
			}
		}
	}
	return result
}

// dumpcp dumps cover profile result to io.Writer.
func dumpcp(w io.Writer, cps []*cover.Profile) {
	if len(cps) == 0 {
		return
	}
	fmt.Fprintf(w, "mode: %v\n", cps[0].Mode)
	for _, cp := range cps {
		for _, b := range cp.Blocks {
			_ = b
			// ref: golang.org/x/tools/cover
			// name.go:line.column,line.column numberOfStatements count
			const blockFmt = "%s:%d.%d,%d.%d %d %d\n"
			fmt.Fprintf(w, blockFmt, cp.FileName, b.StartLine, b.StartCol, b.EndLine, b.EndCol, b.NumStmt, b.Count)
		}
	}
}
