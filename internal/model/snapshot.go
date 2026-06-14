package model

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ReadStatus records whether an evidence source was read successfully.
type ReadStatus string

const (
	ReadOK         ReadStatus = "ok"
	ReadUnreadable ReadStatus = "unreadable"
)

// Condition mirrors metav1.Condition fields the engine cares about.
type Condition struct {
	Type    string
	Status  string // "True" | "False" | "Unknown"
	Reason  string
	Message string
}

// StuckObject is one resource caught mid-deletion.
type StuckObject struct {
	Ref ResourceRef
	// DeletionTimestamp is a plain *time.Time converted from metav1.Time at the
	// discover/snapshot boundary; do not import metav1 into this package.
	DeletionTimestamp   *time.Time
	MetadataFinalizers  []string // JSON path metadata.finalizers
	SpecFinalizers      []string // JSON path spec.finalizers (namespaces)
	OwnerRefs           []ResourceRef
	NamespaceConditions []Condition // populated for namespace targets
	ResourceVersion     string
}

// AllFinalizers returns metadata + spec finalizers combined.
func (o StuckObject) AllFinalizers() []string {
	return append(append([]string{}, o.MetadataFinalizers...), o.SpecFinalizers...)
}

// Snapshot is an immutable, point-in-time read of the cluster.
type Snapshot struct {
	Now               time.Time
	Targets           []StuckObject
	RawAPIServices    []unstructured.Unstructured
	RawCRDs           []unstructured.Unstructured
	RawValidating     []unstructured.Unstructured
	RawMutating       []unstructured.Unstructured
	RawDeployments    []unstructured.Unstructured
	RawStatefulSets   []unstructured.Unstructured
	RawDaemonSets     []unstructured.Unstructured
	RawPods           []unstructured.Unstructured
	RawEndpointSlices []unstructured.Unstructured
	SourceStatus      map[Source]ReadStatus
}

// Readable reports whether the given source was read successfully. Unknown
// sources are treated as not-readable, so missing reads veto a DEAD verdict.
func (s Snapshot) Readable(src Source) bool {
	return s.SourceStatus[src] == ReadOK
}
