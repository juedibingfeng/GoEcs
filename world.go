package ecs

import (
	"reflect"
	"sync"
	"time"
)

// world ECS世界实现
type world struct {
	entityManager    EntityManager
	componentManager ComponentManager
	systemManager    *systemManager
	mu               sync.RWMutex
	running          bool
	lastUpdateTime   time.Time
	totalUpdates     int64
	totalUpdateNano  int64
}

func (w *world) Step(dt time.Duration) error {
	if err := w.Update(dt); err != nil {
		return err
	}
	// 清理孤立组件
	aliveEntities := w.entityManager.AliveEntities()
	aliveMap := make(map[EntityID]bool)
	for _, id := range aliveEntities {
		aliveMap[id] = true
	}

	// 清理死亡实体的组件
	for _, componentType := range w.componentManager.GetRegisteredTypes() {
		type removalTarget struct {
			entityID EntityID
			compType reflect.Type
		}
		toRemove := make([]removalTarget, 0)
		compType := componentType
		_ = w.componentManager.IterateComponents(compType, func(entityID EntityID, component Component) error {
			if !aliveMap[entityID] {
				toRemove = append(toRemove, removalTarget{
					entityID: entityID,
					compType: compType,
				})
			}
			return nil
		})
		for _, target := range toRemove {
			_ = w.componentManager.RemoveComponent(target.entityID, target.compType)
		}
	}

	return nil
}

func (w *world) Destroy() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.running = false

	// 销毁所有系统
	systems := w.systemManager.GetAll()
	for _, system := range systems {
		_ = system.Destroy()
	}

	// 销毁所有实体
	entities := w.entityManager.AliveEntities()
	for _, entityID := range entities {
		_ = w.entityManager.Destroy(entityID)
		_ = w.componentManager.RemoveAllComponents(entityID)
	}

	return nil
}

func (w *world) Stats() WorldStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var avgTime float64
	if w.totalUpdates > 0 {
		avgTime = float64(w.totalUpdateNano) / float64(w.totalUpdates) / 1_000_000.0 // 转换为毫秒
	}

	componentCount := w.componentManager.GetTotalComponentCount()

	return WorldStats{
		TotalEntities:  w.entityManager.GetTotalCreated(),
		AliveEntities:  w.entityManager.GetAliveCount(),
		TotalSystems:   w.systemManager.Count(),
		ComponentTypes: len(w.componentManager.GetRegisteredTypes()),
		ComponentCount: componentCount,
		LastUpdateTime: w.lastUpdateTime.UnixMilli(),
		AverageTime:    avgTime,
	}
}

// NewWorld 创建ECS世界
func NewWorld() World {
	return &world{
		entityManager:    newEntityManager(),
		componentManager: NewComponentManager(),
		systemManager:    newSystemManager(),
		running:          false,
	}
}

// EntityManager 获取实体管理器
func (w *world) EntityManager() EntityManager {
	return w.entityManager
}

// ComponentManager 获取组件管理器
func (w *world) ComponentManager() ComponentManager {
	return w.componentManager
}

// RegisterSystem 注册系统
func (w *world) RegisterSystem(system System) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.systemManager.Register(system); err != nil {
		return err
	}

	return system.Init(w)
}

// UnregisterSystem 注销系统
func (w *world) UnregisterSystem(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.systemManager.Unregister(name)
}

// GetSystem 获取系统
func (w *world) GetSystem(name string) (System, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.systemManager.Get(name)
}

// Update 更新所有系统
func (w *world) Update(dt time.Duration) error {
	w.mu.RLock()
	systems := w.systemManager.GetAll()
	w.mu.RUnlock()

	start := time.Now()

	for _, system := range systems {
		if err := system.Update(dt); err != nil {
			return err
		}
	}

	w.mu.Lock()
	w.totalUpdates++
	w.totalUpdateNano += time.Since(start).Nanoseconds()
	w.lastUpdateTime = time.Now()
	w.mu.Unlock()

	return nil
}

// CreateEntity 创建实体并添加组件
func (w *world) CreateEntity(components ...Component) EntityID {
	entityID := w.EntityManager().Create()
	for _, component := range components {
		_ = w.ComponentManager().AddComponent(entityID, component)
	}
	return entityID
}

// DestroyEntity 销毁实体并清理组件
func (w *world) DestroyEntity(entityID EntityID) error {
	if err := w.EntityManager().Destroy(entityID); err != nil {
		return err
	}
	return w.ComponentManager().RemoveAllComponents(entityID)
}

// AddComponent 安全添加组件
func (w *world) AddComponent(entityID EntityID, component Component) error {
	if !w.entityManager.IsAlive(entityID) {
		return ErrEntityNotFound
	}
	return w.componentManager.AddComponent(entityID, component)
}
