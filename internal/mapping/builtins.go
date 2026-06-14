// Package mapping resolves opaque finalizer strings to owner candidates.
package mapping

type builtinEntry struct {
	Owner      string
	Kind       string
	RarelyDead bool // core control-plane owner: bias toward SLOW/UNKNOWN
}

// builtinTable maps well-known finalizers (verdict-engine.md §5a).
var builtinTable = map[string]builtinEntry{
	"kubernetes":                   {Owner: "namespace controller (kube-controller-manager)", Kind: "Builtin", RarelyDead: true},
	"kubernetes.io/pv-protection":  {Owner: "pv-protection-controller (KCM)", Kind: "Builtin", RarelyDead: true},
	"kubernetes.io/pvc-protection": {Owner: "pvc-protection-controller (KCM)", Kind: "Builtin", RarelyDead: true},
	"foregroundDeletion":           {Owner: "garbage collector (KCM)", Kind: "Builtin", RarelyDead: true},
	"orphan":                       {Owner: "garbage collector (KCM)", Kind: "Builtin", RarelyDead: true},
	"service.kubernetes.io/load-balancer-cleanup": {Owner: "service controller / cloud-provider", Kind: "Builtin", RarelyDead: true},
}

func builtin(finalizer string) (builtinEntry, bool) {
	e, ok := builtinTable[finalizer]
	return e, ok
}
