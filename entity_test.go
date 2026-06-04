package ecs

import (
	"sync"
	"testing"
)

func TestEntityManager_CreateDestroy(t *testing.T) {
	em := newEntityManager()

	id := em.Create()
	if id == NilEntityID {
		t.Fatal("created entity should not be NilEntityID")
	}
	if !em.IsAlive(id) {
		t.Fatal("entity should be alive after creation")
	}

	if err := em.Destroy(id); err != nil {
		t.Fatalf("destroy failed: %v", err)
	}
	if em.IsAlive(id) {
		t.Fatal("entity should not be alive after destroy")
	}
}

func TestEntityManager_DestroyNonExistent(t *testing.T) {
	em := newEntityManager()
	if err := em.Destroy(EntityID(9999)); err != ErrEntityNotFound {
		t.Fatalf("expected ErrEntityNotFound, got %v", err)
	}
}

func TestEntityManager_Counts(t *testing.T) {
	em := newEntityManager()

	id1 := em.Create()
	id2 := em.Create()

	if em.GetAliveCount() != 2 {
		t.Fatalf("expected 2 alive, got %d", em.GetAliveCount())
	}
	if em.GetTotalCreated() != 2 {
		t.Fatalf("expected total 2, got %d", em.GetTotalCreated())
	}

	_ = em.Destroy(id1)

	if em.GetAliveCount() != 1 {
		t.Fatalf("expected 1 alive after destroy, got %d", em.GetAliveCount())
	}
	if em.GetTotalCreated() != 2 {
		t.Fatalf("total should still be 2 after destroy, got %d", em.GetTotalCreated())
	}
	_ = id2
}

func TestEntityManager_MultiWorld_IndependentIDs(t *testing.T) {
	em1 := newEntityManager()
	em2 := newEntityManager()

	id1 := em1.Create()
	_ = em2.Create()

	// 销毁 em1 的实体，不应影响 em2 的存活状态
	_ = em1.Destroy(id1)
	if em1.IsAlive(id1) {
		t.Error("em1: entity should be dead after destroy")
	}
	// em2 创建了自己的实体，其 alive 计数不受 em1 操作影响
	if em2.GetAliveCount() != 1 {
		t.Errorf("em2 should still have 1 alive entity, got %d", em2.GetAliveCount())
	}
}

func TestEntityManager_AliveEntities(t *testing.T) {
	em := newEntityManager()

	id1 := em.Create()
	id2 := em.Create()
	_ = em.Destroy(id1)

	alive := em.AliveEntities()
	if len(alive) != 1 || alive[0] != id2 {
		t.Fatalf("expected [%d], got %v", id2, alive)
	}
}

func TestEntityManager_ConcurrentCreate(t *testing.T) {
	em := newEntityManager()
	const n = 1000

	var wg sync.WaitGroup
	ids := make(chan EntityID, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- em.Create()
		}()
	}
	wg.Wait()
	close(ids)

	seen := make(map[EntityID]bool)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate entity ID: %d", id)
		}
		seen[id] = true
	}

	if em.GetAliveCount() != n {
		t.Fatalf("expected %d alive, got %d", n, em.GetAliveCount())
	}
}
