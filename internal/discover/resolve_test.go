package discover

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTarget(t *testing.T) {
	cases := []struct {
		in      string
		ns      string // default namespace
		group   string
		res     string
		objNS   string
		name    string
		wantErr bool
	}{
		{in: "ns/foo", res: "namespaces", name: "foo"},
		{in: "pvc/data-0", ns: "team-a", res: "pvc", objNS: "team-a", name: "data-0"},
		{in: "widgets.example.com/w1", ns: "team-a", group: "example.com", res: "widgets", objNS: "team-a", name: "w1"},
		{in: "garbage", wantErr: true},
		{in: "/x", wantErr: true},
	}
	for _, c := range cases {
		got, err := ParseTarget(c.in, c.ns)
		if c.wantErr {
			require.Error(t, err, c.in)
			continue
		}
		require.NoError(t, err, c.in)
		assert.Equal(t, c.res, got.Resource, c.in)
		assert.Equal(t, c.group, got.Group, c.in)
		assert.Equal(t, c.name, got.Name, c.in)
		assert.Equal(t, c.objNS, got.Namespace, c.in)
	}
}
