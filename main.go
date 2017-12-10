package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/tools/cover"
)

const usageMessage = "" +
	`Usage:	goverage [flags] -coverprofile=coverage.out package...
`

var (
	coverprofile string
	covermode    string
	cpu          string
	parallel     string
	timeout      string
	short        bool
	v            bool
	x            bool
	race         bool
)

func init() {
	flag.StringVar(&coverprofile, "coverprofile", "coverage.out", "Write a coverage profile to the file after all tests have passed")
	flag.StringVar(&covermode, "covermode", "", "sent as covermode argument to go test")
	flag.StringVar(&cpu, "cpu", "", "sent as cpu argument to go test")
	flag.StringVar(&parallel, "parallel", "", "sent as parallel argument to go test")
	flag.StringVar(&timeout, "timeout", "", "sent as timeout argument to go test")
	flag.BoolVar(&short, "short", false, "sent as short argument to go test")
	flag.BoolVar(&v, "v", false, "sent as v argument to go test")
	flag.BoolVar(&x, "x", false, "sent as x argument to go test")
	flag.BoolVar(&race, "race", false, "enable data race detection")
}

func usage() {
	fmt.Fprintln(os.Stderr, usageMessage)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	os.Exit(2)
}

type ExitError struct {
	Msg  string
	Code int
}

func (e *ExitError) Error() string {
	return e.Msg
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if err := run(coverprofile, flag.Args(), covermode, cpu, parallel, timeout, short, v); err != nil {
		code := 1
		if err, ok := err.(*ExitError); ok {
			code = err.Code
		}
		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(code)
	}
}

func run(coverprofile string, args []string, covermode, cpu, parallel, timeout string, short, v bool) error {
	if coverprofile == "" {
		usage()
		return nil
	}
	if race && covermode != "" && covermode != "atomic" {
		return fmt.Errorf("cannot use race flag and covermode=%s. See more detail on golang/go#12118.", covermode)
	}

	file, err := os.Create(coverprofile)
	if err != nil {
		return err
	}
	defer file.Close()
	// pkgs is packages to run tests and get coverage.
	var pkgs []string
	for _, pkg := range args {
		ps, err := getPkgs(pkg)
		if err != nil {
			return err
		}
		pkgs = append(pkgs, ps...)
	}
	if len(pkgs) == 0 {
		pkgs = []string{"."}
	}
	coverpkg := strings.Join(pkgs, ",")
	optionalArgs := buildOptionalTestArgs(coverpkg, covermode, cpu, parallel, timeout, short, v)
	profiles := make([]string, 0, len(pkgs))
	hasFailedTest := false
	for _, pkg := range pkgs {
		coverprofile, success, err := coverage(pkg, optionalArgs, v)
		if !success {
			hasFailedTest = true
		}
		if err != nil {
			// Do not return err here. It could be just tests are not found for the package.
			log.Printf("got error for package %q: %v", pkg, err)
			continue
		}
		if coverprofile != "" {
			defer os.Remove(coverprofile)
			profiles = append(profiles, coverprofile)
		}
	}
	cp, err := mergeProfiles(profiles, covermode)
	if err != nil {
		return &ExitError{Code: 1, Msg: err.Error()}
	}
	dumpcp(file, cp)
	if hasFailedTest {
		return &ExitError{Code: 1}
	}
	return nil
}

// buildOptionalTestArgs returns common optional args for go test regardless
// target packages. coverpkg must not be empty.
func buildOptionalTestArgs(coverpkg, covermode, cpu, parallel, timeout string, short, v bool) []string {
	args := []string{"-coverpkg", coverpkg}
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
	if x {
		args = append(args, "-x")
	}
	if race {
		args = append(args, "-race")
	}
	return args
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
		if !(strings.Contains(p, "/vendor/") || strings.HasPrefix(p, "vendor/")) {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, nil
}

// coverage runs test for the given pkg and returns cover profile.
// success indicates "go test" succeeded or not. coverage may return profiles
// even when success=false. When "go test" fails, coverage outputs "go test"
// result to stdout even when verbose=false.
// Caller is expected to remove returned coverprofile.
func coverage(pkg string, optArgs []string, verbose bool) (coverprofile string, success bool, err error) {
	coverprofile, err = tmpProfileName()
	if err != nil {
		return "", false, err
	}
	args := append([]string{"test", pkg, "-coverprofile", coverprofile}, optArgs...)
	cmd := exec.Command("go", args...)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}
	if err := cmd.Run(); err != nil {
		fmt.Fprint(os.Stdout, stdout.String())
		fmt.Fprint(os.Stderr, stderr.String())
		// "go test" can creates coverprofile even when "go test" failes, so do not
		// return error here if coverprofile is created.
		if !isExist(coverprofile) {
			return "", false, fmt.Errorf("failed to run 'go test %v': %v", pkg, err)
		}
	} else {
		if !isExist(coverprofile) {
			// There are no test and coverprofile is not created.
			return "", true, nil
		}
		success = true
	}
	return coverprofile, success, err
}

func tmpProfileName() (string, error) {
	f, err := ioutil.TempFile("", "goverage")
	if err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(f.Name()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func isExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func mergeProfiles(profiles []string, covermode string) ([]*cover.Profile, error) {
	mergedProfile, err := ioutil.TempFile("", "goverage-merged-profile")
	if err != nil {
		return nil, err
	}
	defer os.Remove(mergedProfile.Name())
	defer mergedProfile.Close()

	if covermode == "" {
		covermode = "set"
	}

	if _, err = fmt.Fprintf(mergedProfile, "mode: %s\n", covermode); err != nil {
		return nil, err
	}

	expect := fmt.Sprintf("mode: %s\n", covermode)
	for _, file := range profiles {
		buf := make([]byte, len(expect))
		r, err := os.Open(file)
		if err != nil {
			continue
		}
		defer r.Close()

		n, err := io.ReadFull(r, buf)
		if n == 0 {
			continue
		}
		if err != nil || string(buf) != expect {
			return nil, fmt.Errorf("error: test wrote malformed coverage profile: %s", buf)
		}
		_, err = io.Copy(mergedProfile, r)
		if err != nil {
			return nil, fmt.Errorf("error: saving coverage profile: %v", err)
		}
	}

	if err := mergedProfile.Close(); err != nil {
		return nil, err
	}
	return cover.ParseProfiles(mergedProfile.Name())
}

// dumpcp dumps cover profile result to io.Writer.
func dumpcp(w io.Writer, cps []*cover.Profile) {
	if len(cps) == 0 {
		return
	}
	fmt.Fprintf(w, "mode: %v\n", cps[0].Mode)
	for _, cp := range cps {
		for _, b := range cp.Blocks {
			// ref: golang.org/x/tools/cover
			// name.go:line.column,line.column numberOfStatements count
			const blockFmt = "%s:%d.%d,%d.%d %d %d\n"
			fmt.Fprintf(w, blockFmt, cp.FileName, b.StartLine, b.StartCol, b.EndLine, b.EndCol, b.NumStmt, b.Count)
		}
	}
}
