package ecs

import (
	"reflect"
	"sync/atomic"
	"time"
)

// world ECS世界实现
type world struct {
	entityManager    EntityManager
	componentManager ComponentManager
	systemManager    *systemManager

	destroyed       atomic.Bool
	lastUpdateTime  atomic.Int64
	totalUpdates    atomic.Int64
	totalUpdateNano atomic.Int64
	totalCreated    atomic.Int64
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
	w.destroyed.Store(true)

	systems := w.systemManager.GetAll()
	entities := w.entityManager.AliveEntities()

	for _, system := range systems {
		_ = system.Destroy()
	}

	for _, entityID := range entities {
		_ = w.entityManager.Destroy(entityID)
		_ = w.componentManager.RemoveAllComponents(entityID)
	}

	return nil
}

func (w *world) Stats() WorldStats {
	totalUpdates := w.totalUpdates.Load()
	totalUpdateNano := w.totalUpdateNano.Load()

	var avgTime float64
	if totalUpdates > 0 {
		avgTime = float64(totalUpdateNano) / float64(totalUpdates) / 1_000_000.0
	}

	return WorldStats{
		TotalEntities:  int(w.totalCreated.Load()),
		AliveEntities:  w.entityManager.GetAliveCount(),
		TotalSystems:   w.systemManager.Count(),
		ComponentTypes: len(w.componentManager.GetRegisteredTypes()),
		ComponentCount: w.componentManager.GetTotalComponentCount(),
		LastUpdateTime: w.lastUpdateTime.Load(),
		AverageTime:    avgTime,
	}
}

// NewWorld 创建ECS世界
func NewWorld() World {
	return &world{
		entityManager:    newEntityManager(),
		componentManager: NewComponentManager(),
		systemManager:    newSystemManager(),
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
	if err := w.systemManager.Register(system); err != nil {
		return err
	}

	if err := system.Init(w); err != nil {
		_ = w.systemManager.Unregister(system.Name())
		return err
	}

	w.systemManager.MarkInitialized(system.Name())
	return nil
}

// UnregisterSystem 注销系统
func (w *world) UnregisterSystem(name string) error {
	return w.systemManager.Unregister(name)
}

// GetSystem 获取系统
func (w *world) GetSystem(name string) (System, error) {
	return w.systemManager.Get(name)
}

// Update 更新所有系统
func (w *world) Update(dt time.Duration) error {
	if w.destroyed.Load() {
		return nil
	}

	plan := w.systemManager.BuildExecutionPlan()
	start := time.Now()

	var execErr error
	for i, group := range plan {
		if i > 0 && w.destroyed.Load() {
			return nil
		}
		if err := ExecuteGroup(group, dt); err != nil {
			execErr = err
			break
		}
	}

	if execErr != nil {
		return execErr
	}
	w.totalUpdates.Add(1)
	w.totalUpdateNano.Add(time.Since(start).Nanoseconds())
	w.lastUpdateTime.Store(time.Now().UnixMilli())

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
