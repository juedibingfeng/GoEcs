package ecs

import (
	"reflect"
	"testing"
)

func TestQuery_SingleType(t *testing.T) {
	w := NewWorld()
	id1 := w.CreateEntity(testPos{X: 1})
	id2 := w.CreateEntity(testPos{X: 2}, testVel{DX: 3})
	_ = w.CreateEntity(testVel{DX: 5}) // 没有 pos，不应出现

	results := Query(w, reflect.TypeOf(testPos{}))
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	ids := map[EntityID]bool{}
	for _, r := range results {
		ids[r.EntityID] = true
	}
	if !ids[id1] || !ids[id2] {
		t.Fatalf("expected id1 and id2 in results, got %v", ids)
	}
}

func TestQuery_MultiType(t *testing.T) {
	w := NewWorld()
	id := w.CreateEntity(testPos{X: 1}, testVel{DX: 2})
	_ = w.CreateEntity(testPos{X: 3})  // 只有 pos
	_ = w.CreateEntity(testVel{DX: 4}) // 只有 vel

	results := Query(w, reflect.TypeOf(testPos{}), reflect.TypeOf(testVel{}))
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].EntityID != id {
		t.Fatalf("wrong entity: want %d, got %d", id, results[0].EntityID)
	}

	pos := results[0].Components[reflect.TypeOf(testPos{})].(testPos)
	if pos.X != 1 {
		t.Fatalf("wrong X: %f", pos.X)
	}
	vel := results[0].Components[reflect.TypeOf(testVel{})].(testVel)
	if vel.DX != 2 {
		t.Fatalf("wrong DX: %f", vel.DX)
	}
}

func TestQuery_Empty(t *testing.T) {
	w := NewWorld()
	w.CreateEntity(testPos{})

	// 查询不存在的组件类型
	results := Query(w, reflect.TypeOf(testVel{}))
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestQuery_NoTypes(t *testing.T) {
	w := NewWorld()
	w.CreateEntity(testPos{})
	results := Query(w)
	if results != nil {
		t.Fatal("Query with no types should return nil")
	}
}

func TestQueryEach(t *testing.T) {
	w := NewWorld()
	w.CreateEntity(testPos{X: 10}, testVel{DX: 5})
	w.CreateEntity(testPos{X: 20})

	count := 0
	err := QueryEach(w, []reflect.Type{reflect.TypeOf(testPos{}), reflect.TypeOf(testVel{})},
		func(r QueryResult) error {
			count++
			return nil
		})
	if err != nil {
		t.Fatalf("QueryEach failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 callback, got %d", count)
	}
}
