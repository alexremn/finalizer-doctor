package probe

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// probeWorkload looks for a controller workload plausibly matching the owner and
// reports its readiness. Liveness sources (workloads, pods) being unreadable
// yields an Unreadable evidence so the verdict engine can veto DEAD.
func probeWorkload(owner model.OwnerCandidate, snap model.Snapshot) []model.Evidence {
	if !snap.Readable(model.SourceWorkloads) {
		return []model.Evidence{{Signal: "controller workload", Source: model.SourceWorkloads, Observed: "could not read workloads", Class: model.ClassUnreadable}}
	}
	dep, found := matchWorkload(owner, snap.RawDeployments)
	if !found {
		// Absent workload is a dead signal only when the owner was a Workload kind;
		// otherwise it is simply no evidence.
		if owner.Kind == "Workload" {
			return []model.Evidence{{Signal: "controller workload", Source: model.SourceWorkloads, Observed: "no matching controller workload", Class: model.ClassDeadHard, Weight: 35}}
		}
		return nil
	}
	ready, _, _ := unstructured.NestedInt64(dep.Object, "status", "readyReplicas")
	if ready > 0 {
		return []model.Evidence{{Signal: "controller workload", Source: model.SourceWorkloads, Observed: dep.GetName() + " readyReplicas>0", Class: model.ClassLive}}
	}
	// 0 ready: dead only if no running pods either.
	if !snap.Readable(model.SourcePods) {
		return []model.Evidence{{Signal: "controller pods", Source: model.SourcePods, Observed: "could not read pods", Class: model.ClassUnreadable}}
	}
	if len(snap.RawPods) == 0 {
		return []model.Evidence{{Signal: "controller workload", Source: model.SourceWorkloads, Observed: dep.GetName() + " 0 ready, no running pods", Class: model.ClassDeadHard, Weight: 30}}
	}
	return []model.Evidence{{Signal: "controller workload", Source: model.SourceWorkloads, Observed: dep.GetName() + " 0 ready but pods exist", Class: model.ClassLive}}
}

func matchWorkload(owner model.OwnerCandidate, deps []unstructured.Unstructured) (unstructured.Unstructured, bool) {
	for _, d := range deps {
		n := d.GetName()
		if (n != "" && strings.Contains(owner.MatchReason, n)) || strings.HasSuffix(n, "-controller-manager") || strings.HasSuffix(n, "-operator") {
			return d, true
		}
	}
	return unstructured.Unstructured{}, false
}
