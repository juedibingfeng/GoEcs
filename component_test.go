package ecs

import (
	"reflect"
	"sync"
	"testing"
)

// --- 测试用组件 ---

type testPos struct{ X, Y float64 }

func (testPos) Type() reflect.Type { return reflect.TypeOf(testPos{}) }

type testVel struct{ DX, DY float64 }

func (testVel) Type() reflect.Type { return reflect.TypeOf(testVel{}) }

// --- 测试 ---

func TestComponentManager_AddGet(t *testing.T) {
	cm := NewComponentManager()
	id := EntityID(1)

	if err := cm.AddComponent(id, testPos{X: 1, Y: 2}); err != nil {
		t.Fatalf("AddComponent failed: %v", err)
	}

	comp, err := cm.GetComponent(id, reflect.TypeOf(testPos{}))
	if err != nil {
		t.Fatalf("GetComponent failed: %v", err)
	}
	pos := comp.(testPos)
	if pos.X != 1 || pos.Y != 2 {
		t.Fatalf("unexpected component value: %+v", pos)
	}
}

func TestComponentManager_HasComponent(t *testing.T) {
	cm := NewComponentManager()
	id := EntityID(1)

	if cm.HasComponent(id, reflect.TypeOf(testPos{})) {
		t.Fatal("should not have component before add")
	}
	_ = cm.AddComponent(id, testPos{})
	if !cm.HasComponent(id, reflect.TypeOf(testPos{})) {
		t.Fatal("should have component after add")
	}
}

func TestComponentManager_RemoveComponent(t *testing.T) {
	cm := NewComponentManager()
	id := EntityID(1)

	_ = cm.AddComponent(id, testPos{X: 5})
	if err := cm.RemoveComponent(id, reflect.TypeOf(testPos{})); err != nil {
		t.Fatalf("RemoveComponent failed: %v", err)
	}
	if cm.HasComponent(id, reflect.TypeOf(testPos{})) {
		t.Fatal("component should be gone after remove")
	}
}

func TestComponentManager_RemoveNonExistent(t *testing.T) {
	cm := NewComponentManager()
	err := cm.RemoveComponent(EntityID(1), reflect.TypeOf(testPos{}))
	if err != ErrComponentTypeNotRegistered {
		t.Fatalf("expected ErrComponentTypeNotRegistered, got %v", err)
	}

	_ = cm.AddComponent(EntityID(1), testPos{})
	err = cm.RemoveComponent(EntityID(2), reflect.TypeOf(testPos{}))
	if err != ErrComponentNotFound {
		t.Fatalf("expected ErrComponentNotFound, got %v", err)
	}
}

func TestComponentManager_RemoveAllComponents_Idempotent(t *testing.T) {
	cm := NewComponentManager()
	// 对没有任何组件的实体调用 RemoveAll 应该返回 nil
	if err := cm.RemoveAllComponents(EntityID(99)); err != nil {
		t.Fatalf("RemoveAllComponents on empty entity should return nil, got %v", err)
	}
}

func TestComponentManager_RemoveAllComponents(t *testing.T) {
	cm := NewComponentManager()
	id := EntityID(1)

	_ = cm.AddComponent(id, testPos{})
	_ = cm.AddComponent(id, testVel{})

	if err := cm.RemoveAllComponents(id); err != nil {
		t.Fatalf("RemoveAllComponents failed: %v", err)
	}
	if cm.HasComponent(id, reflect.TypeOf(testPos{})) || cm.HasComponent(id, reflect.TypeOf(testVel{})) {
		t.Fatal("components should be gone after RemoveAll")
	}
	if cm.GetTotalComponentCount() != 0 {
		t.Fatalf("total component count should be 0, got %d", cm.GetTotalComponentCount())
	}
}

func TestComponentManager_IterateComponents(t *testing.T) {
	cm := NewComponentManager()
	_ = cm.AddComponent(EntityID(1), testPos{X: 1})
	_ = cm.AddComponent(EntityID(2), testPos{X: 2})

	count := 0
	_ = cm.IterateComponents(reflect.TypeOf(testPos{}), func(id EntityID, c Component) error {
		count++
		return nil
	})
	if count != 2 {
		t.Fatalf("expected 2 iterations, got %d", count)
	}
}

func TestComponentManager_GetEntityComponents(t *testing.T) {
	cm := NewComponentManager()
	id := EntityID(1)
	_ = cm.AddComponent(id, testPos{})
	_ = cm.AddComponent(id, testVel{})

	comps := cm.GetEntityComponents(id)
	if len(comps) != 2 {
		t.Fatalf("expected 2 components, got %d", len(comps))
	}
}

func TestComponentManager_Counts(t *testing.T) {
	cm := NewComponentManager()
	_ = cm.AddComponent(EntityID(1), testPos{})
	_ = cm.AddComponent(EntityID(2), testPos{})
	_ = cm.AddComponent(EntityID(1), testVel{})

	if cm.GetComponentCount(reflect.TypeOf(testPos{})) != 2 {
		t.Fatalf("expected 2 Position components")
	}
	if cm.GetTotalComponentCount() != 3 {
		t.Fatalf("expected total 3, got %d", cm.GetTotalComponentCount())
	}
}

func TestComponentManager_ConcurrentAccess(t *testing.T) {
	cm := NewComponentManager()
	const n = 200

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		id := EntityID(i + 1)
		go func() {
			defer wg.Done()
			_ = cm.AddComponent(id, testPos{X: float64(id)})
		}()
	}
	wg.Wait()

	if cm.GetTotalComponentCount() != n {
		t.Fatalf("expected %d components, got %d", n, cm.GetTotalComponentCount())
	}
}
