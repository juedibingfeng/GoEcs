package ecs

import "errors"

var (
	// ErrEntityNotFound 实体不存在
	ErrEntityNotFound = errors.New("entity not found")

	// ErrComponentNotFound 组件不存在
	ErrComponentNotFound = errors.New("component not found")

	// ErrComponentAlreadyExists 组件已存在
	ErrComponentAlreadyExists = errors.New("component already exists")

	// ErrComponentTypeNotRegistered 组件类型未注册
	ErrComponentTypeNotRegistered = errors.New("component type not registered")

	// ErrSystemNotFound 系统不存在
	ErrSystemNotFound = errors.New("system not found")

	// ErrSystemAlreadyExists 系统已存在
	ErrSystemAlreadyExists = errors.New("system already exists")

	// ErrCircularDependency 循环依赖
	ErrCircularDependency = errors.New("circular dependency detected")
)
