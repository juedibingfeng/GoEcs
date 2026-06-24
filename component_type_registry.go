package ecs

import (
	"errors"
	"reflect"
	"sync"
)

// ComponentTypeID 组件类型 ID
type ComponentTypeID uint32

const MaxComponentTypes = 1024

var ErrMaxComponentTypes = errors.New("ecs: exceeded maximum component types")

// ComponentTypeRegistry 组件类型注册表
type ComponentTypeRegistry struct {
	mu           sync.RWMutex
	typeToID     map[reflect.Type]ComponentTypeID
	idToType     map[ComponentTypeID]reflect.Type
	builtinTypes map[ComponentTypeID]bool
	nextID       ComponentTypeID
}

// globalTypeRegistry 全局单例，将 reflect.Type 映射为整数 ID
var globalTypeRegistry = &ComponentTypeRegistry{
	typeToID:     make(map[reflect.Type]ComponentTypeID),
	idToType:     make(map[ComponentTypeID]reflect.Type),
	builtinTypes: make(map[ComponentTypeID]bool),
	nextID:       1,
}

func RegisterComponentType(componentType reflect.Type) (ComponentTypeID, error) {
	globalTypeRegistry.mu.Lock()
	defer globalTypeRegistry.mu.Unlock()

	if id, exists := globalTypeRegistry.typeToID[componentType]; exists {
		return id, nil
	}
	if globalTypeRegistry.nextID >= MaxComponentTypes {
		return 0, ErrMaxComponentTypes
	}
	id := globalTypeRegistry.nextID
	globalTypeRegistry.nextID++
	globalTypeRegistry.typeToID[componentType] = id
	globalTypeRegistry.idToType[id] = componentType

	return id, nil
}

func GetComponentType(id ComponentTypeID) (reflect.Type, bool) {
	globalTypeRegistry.mu.RLock()
	defer globalTypeRegistry.mu.RUnlock()
	typ, exists := globalTypeRegistry.idToType[id]
	return typ, exists
}
func GetTypeIDCount() int {
	globalTypeRegistry.mu.RLock()
	defer globalTypeRegistry.mu.RUnlock()
	return len(globalTypeRegistry.typeToID)
}
