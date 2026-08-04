package main

import "testing"

func TestSiteLoopbackAddress(t *testing.T) {
	getenv := func(name string) string {
		if name == "SITE_BIND_HOST" {
			return "127.0.0.1"
		}
		return ""
	}
	if got := siteListenAddress(getenv, "8088"); got != "127.0.0.1:8088" {
		t.Fatalf("loopback listen address = %q", got)
	}
	if got := siteListenAddress(func(string) string { return "" }, "8088"); got != ":8088" {
		t.Fatalf("default listen address = %q", got)
	}
}
