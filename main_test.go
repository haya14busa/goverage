package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestRun(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "goverage-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("./example/root"); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if err := run(tmpfile.Name(), []string{"./..."}, "count", "", "", "", false, true); err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	wb, err := ioutil.ReadFile("coverage.ok")
	if err != nil {
		t.Fatal(err)
	}
	want := string(wb)
	if got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}
}
