package ecs

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

// --- 辅助 ---

func newWorld() World {
	return NewWorld()
}

// --- World 测试 ---

func TestWorld_CreateDestroyEntity(t *testing.T) {
	w := newWorld()

	id := w.CreateEntity(testPos{X: 1, Y: 2})
	if !w.EntityManager().IsAlive(id) {
		t.Fatal("entity should be alive")
	}
	if !w.ComponentManager().HasComponent(id, reflect.TypeOf(testPos{})) {
		t.Fatal("entity should have Position component")
	}

	if err := w.DestroyEntity(id); err != nil {
		t.Fatalf("DestroyEntity failed: %v", err)
	}
	if w.EntityManager().IsAlive(id) {
		t.Fatal("entity should be dead after destroy")
	}
	if w.ComponentManager().HasComponent(id, reflect.TypeOf(testPos{})) {
		t.Fatal("components should be removed after entity destroy")
	}
}

func TestWorld_AddComponent_DeadEntity(t *testing.T) {
	w := newWorld()

	id := w.CreateEntity()
	_ = w.DestroyEntity(id)

	err := w.AddComponent(id, testPos{})
	if err != ErrEntityNotFound {
		t.Fatalf("expected ErrEntityNotFound when adding to dead entity, got %v", err)
	}
}

func TestWorld_RegisterSystem(t *testing.T) {
	w := newWorld()
	sys := newTestSystem("S", 10, nil)

	if err := w.RegisterSystem(sys); err != nil {
		t.Fatalf("RegisterSystem failed: %v", err)
	}

	got, err := w.GetSystem("S")
	if err != nil || got.Name() != "S" {
		t.Fatalf("GetSystem failed: %v", err)
	}
}

func TestWorld_Update(t *testing.T) {
	w := newWorld()
	sys := newTestSystem("S", 10, nil)
	_ = w.RegisterSystem(sys)

	if err := w.Update(time.Millisecond * 16); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if sys.updateCount != 1 {
		t.Fatalf("expected updateCount=1, got %d", sys.updateCount)
	}
}

func TestWorld_Step_CleansOrphanComponents(t *testing.T) {
	w := newWorld()

	id := w.CreateEntity(testPos{})
	// 直接通过 entityManager 销毁（绕过 world.DestroyEntity）
	_ = w.EntityManager().Destroy(id)

	// Step 应清理孤立组件
	if err := w.Step(time.Millisecond * 16); err != nil {
		t.Fatalf("Step failed: %v", err)
	}
	if w.ComponentManager().HasComponent(id, reflect.TypeOf(testPos{})) {
		t.Fatal("orphan component should be cleaned up by Step")
	}
}

func TestWorld_Stats(t *testing.T) {
	w := newWorld()
	sys := newTestSystem("S", 10, nil)
	_ = w.RegisterSystem(sys)
	_ = w.CreateEntity(testPos{}, testVel{})
	_ = w.Update(time.Millisecond * 16)

	stats := w.Stats()
	if stats.AliveEntities != 1 {
		t.Errorf("expected 1 alive entity, got %d", stats.AliveEntities)
	}
	if stats.TotalSystems != 1 {
		t.Errorf("expected 1 system, got %d", stats.TotalSystems)
	}
	if stats.ComponentCount != 2 {
		t.Errorf("expected 2 components, got %d", stats.ComponentCount)
	}
	if stats.AverageTime < 0 {
		t.Errorf("average time should be non-negative, got %f", stats.AverageTime)
	}
}

func TestWorld_Destroy(t *testing.T) {
	w := newWorld()
	_ = w.CreateEntity(testPos{})
	_ = w.CreateEntity(testPos{})

	if err := w.Destroy(); err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}
	if w.EntityManager().GetAliveCount() != 0 {
		t.Fatal("all entities should be destroyed")
	}
}

func TestWorld_ConcurrentCreateAndUpdate(t *testing.T) {
	w := newWorld()
	sys := newTestSystem("S", 10, nil)
	_ = w.RegisterSystem(sys)

	const n = 50
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.CreateEntity(testPos{})
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = w.Update(time.Millisecond * 16)
		}()
	}

	wg.Wait()
}

func TestWorld_CircularDependency_PropagatedFromRegister(t *testing.T) {
	w := newWorld()
	_ = w.RegisterSystem(newTestSystem("A", 10, []string{"B"}))
	err := w.RegisterSystem(newTestSystem("B", 5, []string{"A"}))
	if err != ErrCircularDependency {
		t.Fatalf("expected ErrCircularDependency from World.RegisterSystem, got %v", err)
	}
}
