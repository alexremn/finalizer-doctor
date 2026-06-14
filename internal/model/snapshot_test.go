package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSnapshotSourceReadable(t *testing.T) {
	s := Snapshot{
		Now: time.Unix(1000, 0),
		SourceStatus: map[Source]ReadStatus{
			SourceWorkloads:      ReadOK,
			SourceEndpointSlices: ReadUnreadable,
		},
	}
	assert.True(t, s.Readable(SourceWorkloads))
	assert.False(t, s.Readable(SourceEndpointSlices))
	// Unknown source defaults to not-readable (conservative).
	assert.False(t, s.Readable(SourceAPIServices))
}

func TestStuckObjectAllFinalizers(t *testing.T) {
	o := StuckObject{
		MetadataFinalizers: []string{"a", "b"},
		SpecFinalizers:     []string{"kubernetes"},
	}
	assert.ElementsMatch(t, []string{"a", "b", "kubernetes"}, o.AllFinalizers())
}
