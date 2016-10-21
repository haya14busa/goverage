package root

import (
	"github.com/haya14busa/goverage/example/root/sub"
	_ "github.com/haya14busa/vendorpkg"
)

func CoveredFromRoot() {
	_ = "ok"
}

func CoverSub() {
	sub.CoveredFromRoot()
	sub.CoveredFromSubAndRoot()
}
