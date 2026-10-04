package kube

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestToApplication(t *testing.T) {
	u := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"name":   "orders",
			"labels": map[string]any{ServiceLabel: "orders"},
		},
		"status": map[string]any{
			"sync":           map[string]any{"status": "Synced", "revision": "0123456789abcdef"},
			"health":         map[string]any{"status": "Healthy"},
			"operationState": map[string]any{"message": "successfully synced"},
		},
	}}

	got := toApplication(u)
	want := Application{Service: "orders", Sync: "Synced", Health: "Healthy", Revision: "0123456", Message: "successfully synced"}
	if got != want {
		t.Errorf("toApplication() = %+v, want %+v", got, want)
	}
}

func TestToApplicationWithSeveralSources(t *testing.T) {
	u := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"labels": map[string]any{ServiceLabel: "hello"}},
		"status": map[string]any{
			"sync": map[string]any{"status": "Synced", "revisions": []any{"1cc453ab4aeb", "1cc453ab4aeb"}},
		},
	}}
	if got := toApplication(u).Revision; got != "1cc453a" {
		t.Errorf("Revision = %q, want %q", got, "1cc453a")
	}
}
