package api

import "testing"

func TestSessionRegistryAcquireRelease(t *testing.T) {
	r := newSessionRegistry()
	if !r.TryAcquire("iv-1") {
		t.Fatal("first acquire failed")
	}
	if r.TryAcquire("iv-1") {
		t.Fatal("second acquire succeeded")
	}
	if !r.TryAcquire("iv-2") {
		t.Fatal("different key blocked")
	}
	r.Release("iv-1")
	if !r.TryAcquire("iv-1") {
		t.Fatal("acquire after release failed")
	}
	r.Release("iv-1")
	r.Release("iv-1") // idempotent
}
