/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/nethesis/my/backend/logger"
)

// Tunables for the background fan-out; vars so tests can shrink them.
var (
	propagationConcurrency = 10
	propagationTimeout     = 10 * time.Minute
	propagationRetryDelays = []time.Duration{5 * time.Second, 30 * time.Second}
)

// pushTenant is the per-tenant push, swappable in tests.
var pushTenant = RenderAndPushEffective

// Background fan-out states reported by PropagationStatusFor.
const (
	PropagationRunning    = "running"
	PropagationDone       = "done"
	PropagationIncomplete = "incomplete"
)

// PropagationStatus is the progress of the last background fan-out of an
// organization's layer. It lives in memory: a restart forgets it.
type PropagationStatus struct {
	State      string     `json:"state"`
	Total      int        `json:"total"`
	Pushed     int        `json:"pushed"`
	Failed     int        `json:"failed"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// PropagateToTenants re-renders and pushes every tenant with bounded
// concurrency, until ctx expires. It returns one `org <id>: <error>` entry
// per tenant that failed, sorted; empty when all pushes succeeded.
func PropagateToTenants(ctx context.Context, tenants []string) []string {
	failed := pushAll(ctx, tenants, nil)
	warnings := make([]string, 0, len(failed))
	for tenant, err := range failed {
		warnings = append(warnings, fmt.Sprintf("org %s: %s", tenant, err.Error()))
	}
	sort.Strings(warnings)
	return warnings
}

// pushAll pushes tenants concurrently; onPushed, if set, runs after each
// successful push.
func pushAll(ctx context.Context, tenants []string, onPushed func()) map[string]error {
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		failed = map[string]error{}
	)
	sem := make(chan struct{}, propagationConcurrency)
	for _, tenant := range tenants {
		wg.Add(1)
		go func(tenant string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				failed[tenant] = fmt.Errorf("timed out waiting for slot")
				mu.Unlock()
				return
			}
			if err := pushTenant(ctx, tenant); err != nil {
				mu.Lock()
				failed[tenant] = err
				mu.Unlock()
				return
			}
			if onPushed != nil {
				onPushed()
			}
		}(tenant)
	}
	wg.Wait()
	return failed
}

// propagationJob tracks the background fan-out of one organization's layer.
// A save that lands while a run is in flight only marks it pending: the run
// starts over once done, re-reading tenants and layers from the database, so
// N quick saves cost at most two full fan-outs.
type propagationJob struct {
	pending bool
}

var (
	propagationMu       sync.Mutex
	propagationJobs     = map[string]*propagationJob{}
	propagationStatuses = map[string]*PropagationStatus{}
	propagationWG       sync.WaitGroup
)

// PropagationStatusFor returns the status of key's last background fan-out.
func PropagationStatusFor(key string) (PropagationStatus, bool) {
	propagationMu.Lock()
	defer propagationMu.Unlock()
	st, ok := propagationStatuses[key]
	if !ok {
		return PropagationStatus{}, false
	}
	return *st, true
}

func updateStatus(key string, fn func(*PropagationStatus)) {
	propagationMu.Lock()
	defer propagationMu.Unlock()
	st, ok := propagationStatuses[key]
	if !ok {
		st = &PropagationStatus{}
		propagationStatuses[key] = st
	}
	fn(st)
}

// PropagateAsync pushes the tenants returned by listTenants in the
// background, detached from any request, and records its progress under key.
// Each push renders from the database at push time, so a run never writes an
// older layer than the stored one. Tenants that fail are retried after
// propagationRetryDelays; whatever still fails is logged.
func PropagateAsync(key string, listTenants func() ([]string, error)) {
	propagationMu.Lock()
	if job, running := propagationJobs[key]; running {
		job.pending = true
		propagationMu.Unlock()
		return
	}
	propagationStatuses[key] = &PropagationStatus{State: PropagationRunning, StartedAt: time.Now()}
	job := &propagationJob{}
	propagationJobs[key] = job
	propagationWG.Add(1)
	propagationMu.Unlock()

	go func() {
		defer propagationWG.Done()
		defer func() {
			if r := recover(); r != nil {
				logger.Error().Interface("panic", r).Str("org_id", key).Msg("panic while propagating alerting config")
				propagationMu.Lock()
				delete(propagationJobs, key)
				propagationMu.Unlock()
				finishStatus(key, 1)
			}
		}()
		for {
			runPropagation(key, listTenants)
			propagationMu.Lock()
			if !job.pending {
				delete(propagationJobs, key)
				propagationMu.Unlock()
				return
			}
			job.pending = false
			propagationStatuses[key] = &PropagationStatus{State: PropagationRunning, StartedAt: time.Now()}
			propagationMu.Unlock()
		}
	}()
}

// finishStatus closes the current run; a save that arrived meanwhile keeps
// the status running, since the coalesced rerun is about to start.
func finishStatus(key string, failed int) {
	propagationMu.Lock()
	defer propagationMu.Unlock()
	if job, ok := propagationJobs[key]; ok && job.pending {
		return
	}
	st, ok := propagationStatuses[key]
	if !ok {
		return
	}
	now := time.Now()
	st.Failed = failed
	st.FinishedAt = &now
	st.State = PropagationDone
	if failed > 0 {
		st.State = PropagationIncomplete
	}
}

func runPropagation(key string, listTenants func() ([]string, error)) {
	tenants, err := listTenants()
	if err != nil {
		logger.Error().Err(err).Str("org_id", key).Msg("alerting config propagation skipped: cannot list tenants")
		finishStatus(key, 1)
		return
	}
	updateStatus(key, func(st *PropagationStatus) { st.Total = len(tenants) })
	onPushed := func() { updateStatus(key, func(st *PropagationStatus) { st.Pushed++ }) }

	ctx, cancel := context.WithTimeout(context.Background(), propagationTimeout)
	defer cancel()

	start := time.Now()
	todo := tenants
	var failed map[string]error
	for attempt := 0; ; attempt++ {
		failed = pushAll(ctx, todo, onPushed)
		if len(failed) == 0 || attempt >= len(propagationRetryDelays) {
			break
		}
		select {
		case <-time.After(propagationRetryDelays[attempt]):
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
		todo = todo[:0:0]
		for tenant := range failed {
			todo = append(todo, tenant)
		}
	}
	finishStatus(key, len(failed))

	if len(failed) == 0 {
		logger.Info().
			Str("org_id", key).
			Int("tenants", len(tenants)).
			Dur("elapsed", time.Since(start)).
			Msg("alerting config propagated")
		return
	}
	ids := make([]string, 0, len(failed))
	for tenant, err := range failed {
		ids = append(ids, tenant)
		logger.Warn().Err(err).Str("org_id", key).Str("tenant", tenant).Msg("alerting config propagation failed")
	}
	sort.Strings(ids)
	logger.Error().
		Str("org_id", key).
		Int("tenants", len(tenants)).
		Int("failed", len(failed)).
		Strs("failed_tenants", ids).
		Dur("elapsed", time.Since(start)).
		Msg("alerting config propagation incomplete")
}

// WaitPropagations blocks until background fan-outs finish or ctx expires,
// so a graceful shutdown does not cut a fan-out halfway when it can wait.
func WaitPropagations(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		propagationWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
