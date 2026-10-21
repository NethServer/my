/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func stubPush(t *testing.T, push func(ctx context.Context, tenant string) error) {
	t.Helper()
	prevPush, prevDelays := pushTenant, propagationRetryDelays
	pushTenant = push
	propagationRetryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	t.Cleanup(func() {
		pushTenant, propagationRetryDelays = prevPush, prevDelays
	})
}

func waitPropagations(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := WaitPropagations(ctx); err != nil {
		t.Fatalf("propagation did not finish: %v", err)
	}
}

func TestPropagateToTenantsReportsFailures(t *testing.T) {
	stubPush(t, func(_ context.Context, tenant string) error {
		if tenant == "bad" {
			return errors.New("boom")
		}
		return nil
	})
	got := PropagateToTenants(context.Background(), []string{"a", "bad", "b"})
	if len(got) != 1 || got[0] != "org bad: boom" {
		t.Fatalf("warnings = %v", got)
	}
}

func TestPropagateAsyncPushesEveryTenant(t *testing.T) {
	var mu sync.Mutex
	pushed := map[string]int{}
	stubPush(t, func(_ context.Context, tenant string) error {
		mu.Lock()
		pushed[tenant]++
		mu.Unlock()
		return nil
	})
	tenants := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		tenants = append(tenants, string(rune('A'+i)))
	}
	PropagateAsync("org-all", func() ([]string, error) { return tenants, nil })
	waitPropagations(t)
	if len(pushed) != len(tenants) {
		t.Fatalf("pushed %d tenants, want %d", len(pushed), len(tenants))
	}
}

func TestPropagateAsyncRetriesFailedTenants(t *testing.T) {
	var calls atomic.Int32
	stubPush(t, func(_ context.Context, tenant string) error {
		if tenant == "flaky" && calls.Add(1) < 3 {
			return errors.New("mimir busy")
		}
		return nil
	})
	PropagateAsync("org-retry", func() ([]string, error) { return []string{"ok", "flaky"}, nil })
	waitPropagations(t)
	if calls.Load() != 3 {
		t.Fatalf("flaky pushed %d times, want 3", calls.Load())
	}
}

func TestPropagateAsyncCoalescesSaves(t *testing.T) {
	release := make(chan struct{})
	var runs atomic.Int32
	stubPush(t, func(_ context.Context, _ string) error {
		<-release
		return nil
	})
	list := func() ([]string, error) {
		runs.Add(1)
		return []string{"t1"}, nil
	}
	PropagateAsync("org-busy", list)
	for i := 0; i < 5; i++ {
		PropagateAsync("org-busy", list)
	}
	close(release)
	waitPropagations(t)
	if runs.Load() != 2 {
		t.Fatalf("runs = %d, want 2 (first save + one coalesced rerun)", runs.Load())
	}
}

func TestPropagationStatusReportsProgress(t *testing.T) {
	release := make(chan struct{})
	stubPush(t, func(_ context.Context, tenant string) error {
		<-release
		if tenant == "bad" {
			return errors.New("boom")
		}
		return nil
	})
	PropagateAsync("org-status", func() ([]string, error) { return []string{"a", "b", "bad"}, nil })
	if st, ok := PropagationStatusFor("org-status"); !ok || st.State != PropagationRunning {
		t.Fatalf("status while running = %+v, %v", st, ok)
	}
	close(release)
	waitPropagations(t)
	st, _ := PropagationStatusFor("org-status")
	if st.State != PropagationIncomplete || st.Total != 3 || st.Pushed != 2 || st.Failed != 1 || st.FinishedAt == nil {
		t.Fatalf("final status = %+v", st)
	}
}

func TestPropagationStatusDone(t *testing.T) {
	stubPush(t, func(_ context.Context, _ string) error { return nil })
	PropagateAsync("org-done", func() ([]string, error) { return []string{"a", "b"}, nil })
	waitPropagations(t)
	st, _ := PropagationStatusFor("org-done")
	if st.State != PropagationDone || st.Pushed != 2 || st.Failed != 0 {
		t.Fatalf("final status = %+v", st)
	}
	if _, ok := PropagationStatusFor("never-saved"); ok {
		t.Fatal("unexpected status for an org that never propagated")
	}
}
