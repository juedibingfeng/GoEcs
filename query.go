package ecs

import "reflect"

// QueryResult 单个实体的查询结果，包含该实体所有匹配的组件
type QueryResult struct {
	EntityID   EntityID
	Components map[reflect.Type]Component
}

// Query 返回同时拥有所有指定组件类型的实体列表

func Query(world World, componentTypes ...reflect.Type) []QueryResult {
	if len(componentTypes) == 0 {
		return nil
	}

	cm := world.ComponentManager()

	// 从数量最少的组件类型出发，减少初始候选集
	pivot := componentTypes[0]
	for _, t := range componentTypes[1:] {
		if cm.GetComponentCount(t) < cm.GetComponentCount(pivot) {
			pivot = t
		}
	}

	var results []QueryResult
	_ = cm.IterateComponents(pivot, func(entityID EntityID, _ Component) error {
		comps := make(map[reflect.Type]Component, len(componentTypes))
		for _, t := range componentTypes {
			comp, err := cm.GetComponent(entityID, t)
			if err != nil {
				return nil // 缺少任意组件则跳过该实体
			}
			comps[t] = comp
		}
		results = append(results, QueryResult{EntityID: entityID, Components: comps})
		return nil
	})

	return results
}

// QueryEach 对每个同时拥有所有指定组件类型的实体执行回调
func QueryEach(world World, componentTypes []reflect.Type, fn func(QueryResult) error) error {
	if len(componentTypes) == 0 {
		return nil
	}

	cm := world.ComponentManager()

	pivot := componentTypes[0]
	for _, t := range componentTypes[1:] {
		if cm.GetComponentCount(t) < cm.GetComponentCount(pivot) {
			pivot = t
		}
	}

	return cm.IterateComponents(pivot, func(entityID EntityID, _ Component) error {
		comps := make(map[reflect.Type]Component, len(componentTypes))
		for _, t := range componentTypes {
			comp, err := cm.GetComponent(entityID, t)
			if err != nil {
				return nil
			}
			comps[t] = comp
		}
		return fn(QueryResult{EntityID: entityID, Components: comps})
	})
}
