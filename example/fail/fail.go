package fail

import (
	"github.com/haya14busa/goverage/example/fail/sub"
)

func ok() {
	_ = "ok"
	sub.CoveredFromRoot()
}
