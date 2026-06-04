package ecs

import (
	"sync"
	"sync/atomic"
)

// entityManager 实体管理器实现
type entityManager struct {
	alive        map[EntityID]bool
	totalCreated int
	nextID       atomic.Uint64
	mu           sync.RWMutex
}

// newEntityManager 创建实体管理器
func newEntityManager() EntityManager {
	return &entityManager{
		alive: make(map[EntityID]bool),
	}
}

// Create 创建新实体
func (em *entityManager) Create() EntityID {
	id := EntityID(em.nextID.Add(1))

	em.mu.Lock()
	em.alive[id] = true
	em.totalCreated++
	em.mu.Unlock()

	return id
}

// GetTotalCreated 获取累计创建实体数量
func (em *entityManager) GetTotalCreated() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.totalCreated
}

// Destroy 销毁实体
func (em *entityManager) Destroy(entityID EntityID) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.alive[entityID]; !exists {
		return ErrEntityNotFound
	}

	delete(em.alive, entityID)
	return nil
}

// IsAlive 检查实体是否存活
func (em *entityManager) IsAlive(entityID EntityID) bool {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return em.alive[entityID]
}

// GetAliveCount 获取存活实体数量
func (em *entityManager) GetAliveCount() int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return len(em.alive)
}

// AliveEntities 获取所有存活实体
func (em *entityManager) AliveEntities() []EntityID {
	em.mu.RLock()
	defer em.mu.RUnlock()

	entities := make([]EntityID, 0, len(em.alive))
	for id := range em.alive {
		entities = append(entities, id)
	}

	return entities
}
