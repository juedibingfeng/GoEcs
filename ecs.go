package ecs

import (
	"reflect"
	"time"
)

// EntityID 实体组件唯一ID
type EntityID uint64

// 默认给一个特殊实体ID
const NilEntityID EntityID = 0

// 组件接口
type Component interface {
	//组件类型
	Type() reflect.Type
}

// 系统接口
type System interface {
	// Name 系统名字
	Name() string
	// Priority 系统优先级（越大越先执行）
	Priority() int
	// Dependencies 系统依赖的其他系统名称（依赖项保证在本系统之前执行）
	Dependencies() []string
	// Init 初始化系统
	Init(world World) error
	// Update 系统更新逻辑
	Update(dt time.Duration) error
	// Destroy 销毁系统
	Destroy() error
}

// EntityManager 实体管理器接口
type EntityManager interface {
	// Create 创建新实体
	Create() EntityID

	// Destroy 销毁实体
	Destroy(entityID EntityID) error

	// IsAlive 检查实体是否存活
	IsAlive(entityID EntityID) bool

	// GetAliveCount 获取存活实体数量
	GetAliveCount() int

	// GetTotalCreated 获取累计创建实体数量
	GetTotalCreated() int

	// AliveEntities 获取所有存活实体
	AliveEntities() []EntityID
}

// ComponentManager 组件管理器接口
type ComponentManager interface {
	// RegisterComponent 注册组件类型
	RegisterComponent(componentType reflect.Type) error

	// AddComponent 添加组件
	AddComponent(entityID EntityID, component Component) error

	// GetComponent 获取组件
	GetComponent(entityID EntityID, componentType reflect.Type) (Component, error)

	// HasComponent 检查实体是否有某组件
	HasComponent(entityID EntityID, componentType reflect.Type) bool

	// RemoveComponent 移除组件
	RemoveComponent(entityID EntityID, componentType reflect.Type) error

	// RemoveAllComponents 移除实体的所有组件
	RemoveAllComponents(entityID EntityID) error

	// IterateComponents 遍历某种组件的所有实体
	IterateComponents(componentType reflect.Type, callback func(EntityID, Component) error) error

	// GetEntityComponents 获取实体的所有组件
	GetEntityComponents(entityID EntityID) map[reflect.Type]Component

	// GetRegisteredTypes 获取已注册的组件类型
	GetRegisteredTypes() []reflect.Type

	// GetComponentCount 获取某类型组件数量
	GetComponentCount(componentType reflect.Type) int

	// GetTotalComponentCount 获取组件总数
	GetTotalComponentCount() int
}

// World ECS世界接口
type World interface {
	// EntityManager 获取实体管理器
	EntityManager() EntityManager

	// ComponentManager 获取组件管理器
	ComponentManager() ComponentManager

	// RegisterSystem 注册系统
	RegisterSystem(system System) error

	// UnregisterSystem 注销系统
	UnregisterSystem(name string) error

	// GetSystem 获取系统
	GetSystem(name string) (System, error)

	// Update 更新所有系统
	Update(dt time.Duration) error

	// Step 执行一步（更新 + 清理）
	Step(dt time.Duration) error

	// CreateEntity 创建实体并添加组件
	CreateEntity(components ...Component) EntityID

	// DestroyEntity 销毁实体并清理组件
	DestroyEntity(entityID EntityID) error

	// AddComponent 安全添加组件
	AddComponent(entityID EntityID, component Component) error

	// Destroy 销毁世界
	Destroy() error

	// Stats 获取世界统计信息
	Stats() WorldStats
}

// WorldStats 世界统计信息
type WorldStats struct {
	TotalEntities  int     // 总实体数量
	AliveEntities  int     // 存活实体数量
	TotalSystems   int     // 总系统数量
	ComponentTypes int     // 组件类型数量
	ComponentCount int     // 组件总数
	LastUpdateTime int64   // 最后更新时间（毫秒）
	AverageTime    float64 // 平均更新时间（毫秒）
}
