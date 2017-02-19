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

func TestRun_with_test_failed(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "goverage-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("./example/fail"); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	err = run(tmpfile.Name(), []string{"./..."}, "", "", "", "", false, true)
	if err, ok := err.(*ExitError); !ok || err.Code != 1 {
		t.Fatalf("unexpected error: %v", err)
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
