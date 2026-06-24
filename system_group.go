package ecs

import (
	"sync"
	"time"
)

type SystemGroup struct {
	systems  []System
	Parallel bool //组内是否存在交集
}

func (sm *systemManager) BuildExecutionPlan() []SystemGroup {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.orderDirty {
		sm.reorderSystems()
		sm.orderDirty = false
	}
	if len(sm.systemsOrder) == 0 {
		return nil
	}
	depth := sm.calcDependencyDepths()
	maxDepth := 0
	for _, d := range depth {
		if d > maxDepth {
			maxDepth = d
		}
	}
	groups := make([]SystemGroup, 0, maxDepth+1)
	for i := 0; i <= maxDepth; i++ {
		var level []System
		for _, name := range sm.systemsOrder {
			if depth[name] == i && sm.initialized[name] {
				level = append(level, sm.systems[name])
			}
		}
		if len(level) == 0 {
			continue
		}
		groups = append(groups, SystemGroup{
			systems:  level,
			Parallel: len(level) > 1 && sm.canParallelize(level),
		})
	}
	return groups
}
func (sm *systemManager) calcDependencyDepths() map[string]int {
	depth := make(map[string]int, len(sm.systems))
	var calcDepth func(name string) int
	calcDepth = func(name string) int {
		if d, ok := depth[name]; ok {
			return d
		}
		maxDep := 0
		sys, exists := sm.systems[name]
		if !exists {
			depth[name] = 0
			return 0
		}
		for _, dep := range sys.Dependencies() {
			depDep := calcDepth(dep)
			if depDep >= maxDep {
				maxDep = depDep + 1
			}
		}
		depth[name] = maxDep
		return maxDep
	}
	for name := range sm.systems {
		calcDepth(name)
	}
	return depth
}

// canParallelize 检查组内系统的 RequiredComponents 是否无交集
func (sm *systemManager) canParallelize(systems []System) bool {
	compOwners := make(map[string]string)
	for _, sys := range systems {
		for _, ct := range sys.RequiredComponents() {
			typeName := ct.String()
			if _, exists := compOwners[typeName]; exists {
				return false
			}
			compOwners[typeName] = sys.Name()
		}
	}
	return true
}

// ExecuteGroup 执行一个系统组，Parallel 为 true 时并发执行
func ExecuteGroup(group SystemGroup, dt time.Duration) error {
	if !group.Parallel || len(group.systems) <= 1 {
		for _, sys := range group.systems {
			if err := sys.Update(dt); err != nil {
				return err
			}
		}
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(group.systems))

	for _, sys := range group.systems {
		wg.Add(1)
		go func(s System) {
			defer wg.Done()
			if err := s.Update(dt); err != nil {
				errCh <- err
			}
		}(sys)
	}

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok {
		return err
	}
	return nil
}
