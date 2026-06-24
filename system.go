package ecs

import (
	"reflect"
	"sort"
	"sync"
	"time"
)

//系统管理器

type systemManager struct {
	systems      map[string]System
	systemsOrder []string
	initialized  map[string]bool //是否初始化
	orderDirty   bool            //排序延迟
	mu           sync.RWMutex
}

// 创建系统管理器
func newSystemManager() *systemManager {
	return &systemManager{
		systems:      make(map[string]System),
		systemsOrder: make([]string, 0),
		initialized:  make(map[string]bool),
	}
}

// 注册系统
func (sm *systemManager) Register(system System) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	name := system.Name()
	if _, exists := sm.systems[name]; exists {
		return ErrSystemAlreadyExists
	}
	sm.systems[name] = system
	sm.orderDirty = true
	return nil
}

// reorderSystems 按优先级和依赖关系重新排序系统
func (sm *systemManager) reorderSystems() {
	sorted := make([]string, 0, len(sm.systems))
	visited := make(map[string]bool)
	tempVisited := make(map[string]bool)

	systemNames := make([]string, 0, len(sm.systems))
	for name := range sm.systems {
		systemNames = append(systemNames, name)
	}

	for _, name := range systemNames {
		if err := sm.topologicalSort(name, visited, tempVisited, &sorted); err != nil {
			// 存在循环依赖时回退到简单优先级排序
			sort.Slice(systemNames, func(i, j int) bool {
				return sm.systems[systemNames[i]].Priority() > sm.systems[systemNames[j]].Priority()
			})
			sm.systemsOrder = systemNames
			return
		}
	}

	sm.systemsOrder = sm.sortByDependencyLevel(sorted)
}

// sortByDependencyLevel 在保持依赖顺序前提下，对同级系统按优先级排序
func (sm *systemManager) sortByDependencyLevel(sorted []string) []string {
	depth := make(map[string]int)
	var calcDepth func(name string) int
	calcDepth = func(name string) int {
		if d, ok := depth[name]; ok {
			return d
		}
		maxDep := 0
		for _, dep := range sm.systems[name].Dependencies() {
			if _, exists := sm.systems[dep]; exists {
				if d := calcDepth(dep); d >= maxDep {
					maxDep = d + 1
				}
			}
		}
		depth[name] = maxDep
		return maxDep
	}
	for _, name := range sorted {
		calcDepth(name)
	}

	result := make([]string, 0, len(sorted))
	maxDepth := 0
	for _, d := range depth {
		if d > maxDepth {
			maxDepth = d
		}
	}
	for d := 0; d <= maxDepth; d++ {
		level := make([]string, 0)
		for _, name := range sorted {
			if depth[name] == d {
				level = append(level, name)
			}
		}
		sort.Slice(level, func(i, j int) bool {
			return sm.systems[level[i]].Priority() > sm.systems[level[j]].Priority()
		})
		result = append(result, level...)
	}
	return result
}

// 增加一个排序辅助
func (sm *systemManager) topologicalSort(name string, visited, tempVisited map[string]bool, result *[]string) error {
	if tempVisited[name] {
		return ErrCircularDependency
	}

	if visited[name] {
		return nil
	}

	tempVisited[name] = true
	for _, dep := range sm.systems[name].Dependencies() {
		if _, exists := sm.systems[dep]; exists {
			if err := sm.topologicalSort(dep, visited, tempVisited, result); err != nil {
				return err
			}
		}
	}
	delete(tempVisited, name)
	visited[name] = true
	*result = append(*result, name)

	return nil
}

// 注销系统
func (sm *systemManager) Unregister(name string) error {
	sm.mu.Lock()
	system, exists := sm.systems[name]
	if !exists {
		sm.mu.Unlock()
		return ErrSystemNotFound
	}
	delete(sm.systems, name)
	delete(sm.initialized, name)
	sm.orderDirty = true
	sm.mu.Unlock()

	// Destroy 在锁外调用
	_ = system.Destroy()
	return nil
}

// 获取系统
func (sm *systemManager) Get(name string) (System, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	system, exists := sm.systems[name]
	if !exists {
		return nil, ErrSystemNotFound
	}

	return system, nil
}

// GetAll 获取所有系统（按优先级顺序）
func (sm *systemManager) GetAll() []System {
	sm.mu.Lock()
	if sm.orderDirty {
		sm.reorderSystems()
		sm.orderDirty = false
	}
	sm.mu.Unlock()

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	systems := make([]System, 0, len(sm.systemsOrder))
	for _, name := range sm.systemsOrder {
		if sm.initialized[name] {
			systems = append(systems, sm.systems[name])
		}
	}
	return systems
}

// Count 获取系统数量
func (sm *systemManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.systems)
}

// Names 获取所有系统名称
func (sm *systemManager) Names() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	names := make([]string, 0, len(sm.systems))
	for name := range sm.systems {
		names = append(names, name)
	}

	return names
}

type BaseSystem struct {
	world              World
	name               string
	priority           int
	dependencies       []string
	requiredComponents []reflect.Type
}

func NewBaseSystem(name string, priority int, deps []string, requiredComponents []reflect.Type) *BaseSystem {
	return &BaseSystem{
		name:               name,
		priority:           priority,
		dependencies:       deps,
		requiredComponents: requiredComponents,
	}
}

func (bs *BaseSystem) Name() string {
	return bs.name
}

func (bs *BaseSystem) Priority() int {
	return bs.priority
}

func (bs *BaseSystem) Dependencies() []string {
	return bs.dependencies
}

func (bs *BaseSystem) RequiredComponents() []reflect.Type {
	return bs.requiredComponents
}

func (bs *BaseSystem) Init(world World) error {
	bs.world = world
	return nil
}

// 系统更新
func (bs *BaseSystem) Update(dt time.Duration) error {
	//留空交给具体实现
	return nil
}
func (bs *BaseSystem) Destroy() error {
	// 子类实现
	return nil
}

func (bs *BaseSystem) World() World {
	return bs.world
}
func (sm *systemManager) MarkInitialized(name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.initialized[name] = true
}
