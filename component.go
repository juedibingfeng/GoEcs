package ecs

import (
	"fmt"
	"reflect"
	"sync"
)

type componentManager struct {
	mu sync.RWMutex
	//组件类型[实体ID]组件
	componentsByType map[reflect.Type]map[EntityID]Component
	// componentsByEntity [实体ID][组件类型]组件
	componentsByEntity map[EntityID]map[reflect.Type]Component
	//查看是否注册了组件
	registeredTypes map[reflect.Type]bool
}

// 创建组件管理器
func NewComponentManager() ComponentManager {
	return &componentManager{
		componentsByType:   make(map[reflect.Type]map[EntityID]Component),
		componentsByEntity: make(map[EntityID]map[reflect.Type]Component),
		registeredTypes:    make(map[reflect.Type]bool),
	}
}

//实现组件管理器的接口

// 注册组件的类型
func (cm *componentManager) RegisterComponent(componentType reflect.Type) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.registeredTypes[componentType] {
		return ErrComponentAlreadyExists
	}
	cm.registeredTypes[componentType] = true
	cm.componentsByType[componentType] = make(map[EntityID]Component)
	return nil
}

// 添加组件
func (cm *componentManager) AddComponent(entityID EntityID, component Component) error {
	componentType := component.Type()

	cm.mu.Lock()
	defer cm.mu.Unlock()
	//如果没有注册该组件则自动帮其注册
	if !cm.registeredTypes[componentType] {
		cm.registeredTypes[componentType] = true
		cm.componentsByType[componentType] = make(map[EntityID]Component)
	}

	cm.componentsByType[componentType][entityID] = component

	//实体储存
	if _, exists := cm.componentsByEntity[entityID]; !exists {
		cm.componentsByEntity[entityID] = make(map[reflect.Type]Component)
	}
	cm.componentsByEntity[entityID][componentType] = component
	return nil
}

// 获取组件
func (cm *componentManager) GetComponent(entityID EntityID, componentType reflect.Type) (Component, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.registeredTypes[componentType] {
		return nil, ErrComponentTypeNotRegistered
	}

	component, exists := cm.componentsByType[componentType][entityID]
	if !exists {
		return nil, ErrComponentNotFound
	}

	return component, nil
}

// 检查某个实体是否有某组件
func (cm *componentManager) HasComponent(entityID EntityID, componentType reflect.Type) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.registeredTypes[componentType] {
		return false
	}

	_, exists := cm.componentsByType[componentType][entityID]
	return exists
}

// 移除组件
func (cm *componentManager) RemoveComponent(entityID EntityID, componentType reflect.Type) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if !cm.registeredTypes[componentType] {
		return ErrComponentTypeNotRegistered
	}

	if _, exists := cm.componentsByType[componentType][entityID]; !exists {
		return ErrComponentNotFound
	}
	delete(cm.componentsByType[componentType], entityID)

	//从实体中删除该组件
	if components, exists := cm.componentsByEntity[entityID]; exists {
		delete(components, componentType)
		if len(components) == 0 {
			delete(cm.componentsByEntity, entityID)
		}
	}
	return nil
}

// 移除实体所有组件
func (cm *componentManager) RemoveAllComponents(entityID EntityID) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	components, exists := cm.componentsByEntity[entityID]
	if !exists {
		return ErrComponentNotFound
	}

	// 从类型映射中删除所有组件
	for componentType := range components {
		delete(cm.componentsByType[componentType], entityID)
	}

	// 删除实体映射
	delete(cm.componentsByEntity, entityID)

	return nil
}

// 遍历某种组件的所有实体
func (cm *componentManager) IterateComponents(componentType reflect.Type, callback func(EntityID, Component) error) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.registeredTypes[componentType] {
		return ErrComponentTypeNotRegistered
	}

	for entityID, component := range cm.componentsByType[componentType] {
		if err := callback(entityID, component); err != nil {
			return err
		}
	}

	return nil
}

// 获取实体的所有组件
func (cm *componentManager) GetEntityComponents(entityID EntityID) map[reflect.Type]Component {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	components, exists := cm.componentsByEntity[entityID]
	if !exists {
		return nil
	}
	result := make(map[reflect.Type]Component, len(components))
	for k, v := range components {
		result[k] = v
	}

	return result
}

// 获取已注册的组件类型
func (cm *componentManager) GetRegisteredTypes() []reflect.Type {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	types := make([]reflect.Type, 0, len(cm.registeredTypes))
	for t := range cm.registeredTypes {
		types = append(types, t)
	}

	return types
}

// 获取某类型的组件数量
func (cm *componentManager) GetComponentCount(componentType reflect.Type) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.registeredTypes[componentType] {
		return 0
	}

	return len(cm.componentsByType[componentType])
}

// 获取组件总数
func (cm *componentManager) GetTotalComponentCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	total := 0
	for _, components := range cm.componentsByType {
		total += len(components)
	}
	return total
}

// 获取组件的类型
func TypeOf(component Component) reflect.Type {
	t := reflect.TypeOf(component)
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

// BaseComponent 基础组件实现，嵌入后需自行实现 Type() 方法
type BaseComponent struct{}

// 获取组件值的通用方法
func GetComponentValue[T Component](component Component) (T, error) {
	var zero T
	if component == nil {
		return zero, fmt.Errorf("component is nil")
	}

	value, ok := component.(T)
	if !ok {
		return zero, fmt.Errorf("component type mismatch")
	}

	return value, nil
}
