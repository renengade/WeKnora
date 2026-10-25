package operations

import (
	"strings"
	"sync"

	docs "github.com/Tencent/WeKnora/docs"
)

var (
	builtinOnce    sync.Once
	builtinCatalog *Catalog
	builtinErr     error
)

// Builtin returns the catalog compiled from the Swagger document embedded in
// the binary: every operation annotated with x-agent. It panics on an invalid
// declaration, which tests catch before release.
func Builtin() *Catalog {
	builtinOnce.Do(func() {
		var ops []Operation
		ops, builtinErr = FromSpec([]byte(docs.SwaggerInfo.ReadDoc()))
		if builtinErr == nil {
			ops = withoutRemovedSkillOperations(ops)
			builtinCatalog, builtinErr = NewCatalog(ops...)
		}
	})
	if builtinErr != nil {
		panic(builtinErr)
	}
	return builtinCatalog
}

// withoutRemovedSkillOperations drops skill install/management operations whose
// HTTP routes this fork no longer serves (MGI build). The generated Swagger
// document still describes them, and the router asserts at startup that every
// operation maps to a served route.
func withoutRemovedSkillOperations(ops []Operation) []Operation {
	kept := make([]Operation, 0, len(ops))
	for _, op := range ops {
		if strings.HasPrefix(op.Route, "/api/v1/skills/catalog") ||
			strings.Contains(op.Route, "/sandbox-configs/:id/skills") {
			continue
		}
		kept = append(kept, op)
	}
	return kept
}
