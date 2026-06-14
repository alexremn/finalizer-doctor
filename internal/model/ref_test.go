package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestResourceRefString(t *testing.T) {
	r := ResourceRef{
		GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
		Namespace: "",
		Name:      "foo",
	}
	assert.Equal(t, "namespaces/foo", r.String())

	r2 := ResourceRef{
		GVR:       schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"},
		Namespace: "team-a",
		Name:      "w1",
	}
	assert.Equal(t, "team-a/widgets.example.com/w1", r2.String())
}
