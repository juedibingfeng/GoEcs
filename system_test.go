package ecs

import (
	"reflect"
	"testing"
	"time"
)

// --- 测试用系统 ---

type testSystem struct {
	BaseSystem
	updateCount int
}

func newTestSystem(name string, priority int, deps []string) *testSystem {
	return &testSystem{
		BaseSystem: *NewBaseSystem(name, priority, deps, nil),
	}
}

func (s *testSystem) Update(dt time.Duration) error {
	s.updateCount++
	return nil
}

// --- 测试 ---

func TestSystemManager_RegisterGet(t *testing.T) {
	sm := newSystemManager()
	sys := newTestSystem("A", 10, nil)

	if err := sm.Register(sys); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, err := sm.Get("A")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name() != "A" {
		t.Fatalf("unexpected name: %s", got.Name())
	}
}

func TestSystemManager_RegisterDuplicate(t *testing.T) {
	sm := newSystemManager()
	_ = sm.Register(newTestSystem("A", 10, nil))

	if err := sm.Register(newTestSystem("A", 5, nil)); err != ErrSystemAlreadyExists {
		t.Fatalf("expected ErrSystemAlreadyExists, got %v", err)
	}
}

func TestSystemManager_Unregister(t *testing.T) {
	sm := newSystemManager()
	_ = sm.Register(newTestSystem("A", 10, nil))
	_ = sm.Register(newTestSystem("B", 5, nil))

	if err := sm.Unregister("A"); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}
	if sm.Count() != 1 {
		t.Fatalf("expected 1 system after unregister, got %d", sm.Count())
	}
	if _, err := sm.Get("A"); err != ErrSystemNotFound {
		t.Fatal("A should be gone")
	}
}

func TestSystemManager_UnregisterNonExistent(t *testing.T) {
	sm := newSystemManager()
	if err := sm.Unregister("missing"); err != ErrSystemNotFound {
		t.Fatalf("expected ErrSystemNotFound, got %v", err)
	}
}

func TestSystemManager_PriorityOrder(t *testing.T) {
	sm := newSystemManager()
	_ = sm.Register(newTestSystem("low", 1, nil))
	_ = sm.Register(newTestSystem("high", 100, nil))
	_ = sm.Register(newTestSystem("mid", 50, nil))

	all := sm.GetAll()
	if len(all) != 3 {
		t.Fatalf("expected 3 systems")
	}
	// 高优先级先执行
	if all[0].Name() != "high" || all[1].Name() != "mid" || all[2].Name() != "low" {
		names := make([]string, len(all))
		for i, s := range all {
			names[i] = s.Name()
		}
		t.Fatalf("wrong order: %v", names)
	}
}

func TestSystemManager_DependencyOrder(t *testing.T) {
	sm := newSystemManager()
	// B 依赖 A，所以 A 必须在 B 之前执行（即使 B 优先级更高）
	_ = sm.Register(newTestSystem("A", 5, nil))
	_ = sm.Register(newTestSystem("B", 10, []string{"A"}))

	all := sm.GetAll()
	if all[0].Name() != "A" || all[1].Name() != "B" {
		t.Fatalf("dependency not respected: %s before %s", all[0].Name(), all[1].Name())
	}
}

func TestSystemManager_CircularDependency(t *testing.T) {
	sm := newSystemManager()
	_ = sm.Register(newTestSystem("A", 10, []string{"B"}))
	err := sm.Register(newTestSystem("B", 5, []string{"A"}))

	if err != ErrCircularDependency {
		t.Fatalf("expected ErrCircularDependency, got %v", err)
	}
	// B 注册失败后系统列表中只有 A
	if sm.Count() != 1 {
		t.Fatalf("expected 1 system after failed register, got %d", sm.Count())
	}
}

func TestBaseSystem_Fields(t *testing.T) {
	deps := []string{"X"}
	comps := []reflect.Type{reflect.TypeOf(testPos{})}
	bs := NewBaseSystem("test", 42, deps, comps)

	if bs.Name() != "test" {
		t.Errorf("wrong name")
	}
	if bs.Priority() != 42 {
		t.Errorf("wrong priority")
	}
	if len(bs.Dependencies()) != 1 || bs.Dependencies()[0] != "X" {
		t.Errorf("wrong deps")
	}
	if len(bs.RequiredComponents()) != 1 {
		t.Errorf("wrong required components")
	}
}
