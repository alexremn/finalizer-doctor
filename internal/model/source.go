package model

// Source identifies an evidence source so reads can be tracked individually.
type Source string

const (
	SourceTargets        Source = "targets"
	SourceAPIServices    Source = "apiservices"
	SourceCRDs           Source = "crds"
	SourceWebhooks       Source = "webhooks"
	SourceWorkloads      Source = "workloads"
	SourcePods           Source = "pods"
	SourceEndpointSlices Source = "endpointslices"
	SourceDiscovery      Source = "discovery"
)
