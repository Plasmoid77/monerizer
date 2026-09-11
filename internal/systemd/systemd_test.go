package systemd

import (
	"context"
	"testing"
)

const showOut = "Id=a.service\nLoadState=loaded\nActiveState=active\nAfter=b.service sysinit.target\n\nId=b.service\nLoadState=not-found\nActiveState=inactive\n"

func TestShow(t *testing.T) {
	var gotArgs []string
	run := func(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
		gotArgs = append([]string{name}, args...)
		return []byte(showOut), nil, nil
	}
	res, err := Show(context.Background(), run, "a.service", "b.service")
	if err != nil {
		t.Fatal(err)
	}
	if res["a.service"]["ActiveState"] != "active" || res["b.service"]["LoadState"] != "not-found" {
		t.Fatalf("unexpected %v", res)
	}
	if !res["a.service"].HasDependency("After", "b.service") || res["a.service"].HasDependency("After", "b") {
		t.Fatal("HasDependency")
	}
	if gotArgs[0] != "systemctl" || gotArgs[len(gotArgs)-3] != "--" {
		t.Fatalf("args %v", gotArgs)
	}
}
