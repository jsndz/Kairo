package collaboration_test

import (
	"reflect"
	"slices"
	"sync"
	"testing"

	"server/internal/app/collaboration"
)

func TestNewEngine(t *testing.T) {
	eng := collaboration.New()
	if eng == nil {
		t.Fatal("expected non-nil engine from New()")
	}

	owners := eng.AllOwners()
	if len(owners) != 0 {
		t.Fatalf("expected empty owners map, got %v", owners)
	}

	eng2 := collaboration.NewEngine()
	if eng2 == nil {
		t.Fatal("expected non-nil engine from NewEngine()")
	}
}

func TestClaim_BasicAndIdempotent(t *testing.T) {
	eng := collaboration.New()

	// 1. Initial claim on unassigned clips
	granted, rejected := eng.Claim("client-a", []string{"clip-1", "clip-2"})
	if !reflect.DeepEqual(granted, []string{"clip-1", "clip-2"}) {
		t.Fatalf("expected [clip-1, clip-2] granted, got %v", granted)
	}
	if len(rejected) != 0 {
		t.Fatalf("expected empty rejected, got %v", rejected)
	}

	// 2. Re-claim by same owner should be idempotent and granted
	granted2, rejected2 := eng.Claim("client-a", []string{"clip-1"})
	if !reflect.DeepEqual(granted2, []string{"clip-1"}) {
		t.Fatalf("expected [clip-1] granted on re-claim, got %v", granted2)
	}
	if len(rejected2) != 0 {
		t.Fatalf("expected empty rejected, got %v", rejected2)
	}

	// 3. Verify owner lookup
	owner, ok := eng.GetOwner("clip-1")
	if !ok || owner != "client-a" {
		t.Fatalf("expected owner client-a, got %s (ok=%v)", owner, ok)
	}
}

func TestClaim_ConflictAndRejection(t *testing.T) {
	eng := collaboration.New()

	// Client A claims clip-1
	eng.Claim("client-a", []string{"clip-1"})

	// Client B attempts to claim clip-1 and clip-2
	granted, rejected := eng.Claim("client-b", []string{"clip-1", "clip-2"})
	if !reflect.DeepEqual(granted, []string{"clip-2"}) {
		t.Fatalf("expected [clip-2] granted to client-b, got %v", granted)
	}
	if !reflect.DeepEqual(rejected, []string{"clip-1"}) {
		t.Fatalf("expected [clip-1] rejected for client-b, got %v", rejected)
	}

	// Client A still owns clip-1; Client B owns clip-2
	owner1, _ := eng.GetOwner("clip-1")
	if owner1 != "client-a" {
		t.Fatalf("expected clip-1 owner client-a, got %s", owner1)
	}
	owner2, _ := eng.GetOwner("clip-2")
	if owner2 != "client-b" {
		t.Fatalf("expected clip-2 owner client-b, got %s", owner2)
	}
}

func TestClaim_SimultaneousRaceCondition(t *testing.T) {
	eng := collaboration.New()
	targetClip := "contested-clip"

	const concurrency = 20
	var wg sync.WaitGroup
	wg.Add(concurrency)

	type result struct {
		clientID string
		granted  bool
	}
	results := make([]result, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			cid := string(rune('A' + idx))
			g, _ := eng.Claim(cid, []string{targetClip})
			results[idx] = result{
				clientID: cid,
				granted:  slices.Contains(g, targetClip),
			}
		}(i)
	}

	wg.Wait()

	// Exactly one client must have been granted the lock
	grantCount := 0
	var winner string
	for _, res := range results {
		if res.granted {
			grantCount++
			winner = res.clientID
		}
	}

	if grantCount != 1 {
		t.Fatalf("expected exactly 1 winner in simultaneous claim race, got %d", grantCount)
	}

	actualOwner, ok := eng.GetOwner(targetClip)
	if !ok || actualOwner != winner {
		t.Fatalf("expected winner %s to match actual owner %s", winner, actualOwner)
	}
}

func TestRelease_Voluntary(t *testing.T) {
	eng := collaboration.New()

	eng.Claim("client-a", []string{"clip-1", "clip-2"})

	// Client B cannot release clips owned by Client A
	released := eng.Release("client-b", []string{"clip-1"})
	if len(released) != 0 {
		t.Fatalf("expected 0 released by non-owner, got %v", released)
	}

	// Client A releases clip-1
	released = eng.Release("client-a", []string{"clip-1"})
	if !reflect.DeepEqual(released, []string{"clip-1"}) {
		t.Fatalf("expected [clip-1] released, got %v", released)
	}

	// clip-1 should now be unassigned
	_, ok := eng.GetOwner("clip-1")
	if ok {
		t.Fatal("expected clip-1 to be unassigned after release")
	}

	// Client B can now claim clip-1
	granted, rejected := eng.Claim("client-b", []string{"clip-1"})
	if !reflect.DeepEqual(granted, []string{"clip-1"}) || len(rejected) != 0 {
		t.Fatalf("expected client-b to claim released clip-1, got granted=%v, rejected=%v", granted, rejected)
	}
}

func TestReleaseClient_Reclamation(t *testing.T) {
	eng := collaboration.New()

	eng.Claim("client-a", []string{"c1", "c2", "c3"})
	eng.Claim("client-b", []string{"c4"})

	// Reclaim all locks held by client-a
	released := eng.ReleaseClient("client-a")
	if !reflect.DeepEqual(released, []string{"c1", "c2", "c3"}) {
		t.Fatalf("expected [c1, c2, c3] released, got %v", released)
	}

	// Verify client-a's locks are gone
	for _, id := range []string{"c1", "c2", "c3"} {
		if _, ok := eng.GetOwner(id); ok {
			t.Fatalf("expected %s to be released", id)
		}
	}

	// Client B's lock remains intact
	owner, ok := eng.GetOwner("c4")
	if !ok || owner != "client-b" {
		t.Fatalf("expected c4 to remain owned by client-b, got %s (ok=%v)", owner, ok)
	}

	// Releasing non-existent client returns empty
	empty := eng.ReleaseClient("non-existent")
	if len(empty) != 0 {
		t.Fatalf("expected empty slice, got %v", empty)
	}
}

func TestTransfer(t *testing.T) {
	eng := collaboration.New()

	eng.Claim("client-a", []string{"clip-1"})

	// Transfer to client-b
	if err := eng.Transfer("clip-1", "client-b"); err != nil {
		t.Fatalf("unexpected error on Transfer: %v", err)
	}

	owner, ok := eng.GetOwner("clip-1")
	if !ok || owner != "client-b" {
		t.Fatalf("expected owner client-b after transfer, got %s", owner)
	}

	// Validation on empty params
	if err := eng.Transfer("", "client-b"); err == nil {
		t.Fatal("expected error on empty clipID")
	}
	if err := eng.Transfer("clip-1", ""); err == nil {
		t.Fatal("expected error on empty toClient")
	}
}

func TestIsAllowed(t *testing.T) {
	eng := collaboration.New()

	// Unassigned clips are allowed for anyone
	if !eng.IsAllowed("client-a", "unassigned-clip") {
		t.Fatal("expected unassigned clip to be allowed for client-a")
	}
	if !eng.IsAllowed("client-b", "unassigned-clip") {
		t.Fatal("expected unassigned clip to be allowed for client-b")
	}

	// Claim clip
	eng.Claim("client-a", []string{"clip-1"})

	// Allowed for owner
	if !eng.IsAllowed("client-a", "clip-1") {
		t.Fatal("expected clip-1 to be allowed for owner client-a")
	}

	// Denied for peer
	if eng.IsAllowed("client-b", "clip-1") {
		t.Fatal("expected clip-1 to be denied for peer client-b")
	}
}

func TestAllOwners_Immutability(t *testing.T) {
	eng := collaboration.New()
	eng.Claim("client-a", []string{"c1"})

	owners := eng.AllOwners()
	if owners["c1"] != "client-a" {
		t.Fatalf("expected c1=client-a, got %v", owners)
	}

	// Mutating the returned map should not affect internal state
	owners["c1"] = "mutated"
	owners["c2"] = "added"

	fresh := eng.AllOwners()
	if fresh["c1"] != "client-a" {
		t.Fatalf("expected internal state intact, got %s", fresh["c1"])
	}
	if _, ok := fresh["c2"]; ok {
		t.Fatal("expected c2 not present in fresh snapshot")
	}
}
