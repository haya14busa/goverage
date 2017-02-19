package fail

import "testing"

func TestOk(t *testing.T) {
	ok()
	t.Fatal("failed")
}
