package tonic_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/martinvuyk/gadgeto/tonic"
)

func TestRoute_GetTags(t *testing.T) {
	r := &tonic.Route{
		RouteInfo: gin.RouteInfo{
			Method: "GET",
			Path:   "/foo/bar",
		},
	}
	tags := r.GetTags()
	if len(tags) != 1 {
		t.Fatalf("expected to have 1 tag, but got %d", len(tags))
	}
	if tags[0] != "foo" {
		t.Fatalf("expected to have tag='foo', but got tag=%s", tags[0])
	}
	tonic.Tags([]string{"otherTag"})(r)
	tags = r.GetTags()
	if len(tags) != 1 {
		t.Fatalf("expected to have 1 tag, but got %d", len(tags))
	}
	if tags[0] != "otherTag" {
		t.Fatalf("expected to have tag='otherTag', but got tag=%s", tags[0])
	}
	tonic.Tags([]string{"otherTag1", "otherTag2"})(r)
	tags = r.GetTags()
	if len(tags) != 2 {
		t.Fatalf("expected to have 2 tags, but got %d", len(tags))
	}
	if tags[0] != "otherTag1" {
		t.Fatalf("expected to have tag='otherTag1', but got tag=%s", tags[0])
	}
	if tags[1] != "otherTag2" {
		t.Fatalf("expected to have tag='otherTag2', but got tag=%s", tags[0])
	}
}
