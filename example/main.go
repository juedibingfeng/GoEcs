package main

import (
	"fmt"
	"reflect"
	"time"

	"ecsgo"
)

// ==================== 组件定义 ====================

// Position 位置组件
type Position struct {
	X, Y float64
}

func (Position) Type() reflect.Type { return reflect.TypeOf(Position{}) }

// Velocity 速度组件
type Velocity struct {
	DX, DY float64
}

func (Velocity) Type() reflect.Type { return reflect.TypeOf(Velocity{}) }

// Health 生命值组件
type Health struct {
	Current, Max int
}

func (Health) Type() reflect.Type { return reflect.TypeOf(Health{}) }

// Name 名称组件
type Name struct {
	Value string
}

func (Name) Type() reflect.Type { return reflect.TypeOf(Name{}) }

// ==================== 系统定义 ====================

// MovementSystem 移动系统：根据速度更新位置
type MovementSystem struct {
	ecs.BaseSystem
}

func NewMovementSystem() *MovementSystem {
	return &MovementSystem{
		BaseSystem: *ecs.NewBaseSystem(
			"MovementSystem",
			10, // 优先级
			nil,
			[]reflect.Type{reflect.TypeOf(Position{}), reflect.TypeOf(Velocity{})},
		),
	}
}

func (s *MovementSystem) Update(dt time.Duration) error {
	world := s.World()
	sec := dt.Seconds()

	// 先收集需要更新的实体，避免在迭代期间修改组件（会导致死锁）
	type update struct {
		entityID ecs.EntityID
		pos      Position
	}
	var updates []update

	_ = world.ComponentManager().IterateComponents(
		reflect.TypeOf(Position{}),
		func(entityID ecs.EntityID, comp ecs.Component) error {
			pos := comp.(Position)
			velComp, err := world.ComponentManager().GetComponent(entityID, reflect.TypeOf(Velocity{}))
			if err != nil {
				return nil
			}
			vel := velComp.(Velocity)
			pos.X += vel.DX * sec
			pos.Y += vel.DY * sec
			updates = append(updates, update{entityID, pos})
			return nil
		},
	)

	// 迭代结束后再应用更新
	for _, u := range updates {
		_ = world.ComponentManager().AddComponent(u.entityID, u.pos)
	}
	return nil
}

// HealthSystem 生命值系统：检查并移除死亡实体
type HealthSystem struct {
	ecs.BaseSystem
}

func NewHealthSystem() *HealthSystem {
	return &HealthSystem{
		BaseSystem: *ecs.NewBaseSystem(
			"HealthSystem",
			20,
			nil,
			[]reflect.Type{reflect.TypeOf(Health{})},
		),
	}
}

func (s *HealthSystem) Update(dt time.Duration) error {
	world := s.World()
	var toDestroy []ecs.EntityID

	_ = world.ComponentManager().IterateComponents(
		reflect.TypeOf(Health{}),
		func(entityID ecs.EntityID, comp ecs.Component) error {
			hp := comp.(Health)
			if hp.Current <= 0 {
				toDestroy = append(toDestroy, entityID)
			}
			return nil
		},
	)

	for _, id := range toDestroy {
		name := getEntityName(world, id)
		fmt.Printf("  [HealthSystem] %s 死亡，销毁实体 %d\n", name, id)
		_ = world.DestroyEntity(id)
	}
	return nil
}

// DamageSystem 伤害系统：对实体造成伤害
type DamageSystem struct {
	ecs.BaseSystem
	damageQueue []DamageEvent
}

type DamageEvent struct {
	EntityID ecs.EntityID
	Amount   int
}

func NewDamageSystem() *DamageSystem {
	return &DamageSystem{
		BaseSystem: *ecs.NewBaseSystem("DamageSystem", 15, []string{"MovementSystem"}, nil),
	}
}

func (s *DamageSystem) QueueDamage(entityID ecs.EntityID, amount int) {
	s.damageQueue = append(s.damageQueue, DamageEvent{EntityID: entityID, Amount: amount})
}

func (s *DamageSystem) Update(dt time.Duration) error {
	world := s.World()
	for _, ev := range s.damageQueue {
		comp, err := world.ComponentManager().GetComponent(ev.EntityID, reflect.TypeOf(Health{}))
		if err != nil {
			continue
		}
		hp := comp.(Health)
		hp.Current -= ev.Amount
		name := getEntityName(world, ev.EntityID)
		fmt.Printf("  [DamageSystem] %s 受到 %d 点伤害 (HP: %d/%d)\n", name, ev.Amount, hp.Current, hp.Max)
		_ = world.ComponentManager().AddComponent(ev.EntityID, hp)
	}
	s.damageQueue = nil
	return nil
}

// ==================== 辅助函数 ====================

func getEntityName(world ecs.World, entityID ecs.EntityID) string {
	comp, err := world.ComponentManager().GetComponent(entityID, reflect.TypeOf(Name{}))
	if err != nil {
		return "Unknown"
	}
	return comp.(Name).Value
}

// ==================== 主函数 ====================

func main() {
	fmt.Println("=== ecsgo 示例 ===\n")

	// 1. 创建世界
	world := ecs.NewWorld()

	// 2. 注册系统
	damageSys := NewDamageSystem()
	world.RegisterSystem(NewMovementSystem())
	world.RegisterSystem(damageSys)
	world.RegisterSystem(NewHealthSystem())

	// 3. 创建实体
	player := world.CreateEntity(
		Position{X: 0, Y: 0},
		Velocity{DX: 5, DY: 2},
		Health{Current: 100, Max: 100},
		Name{Value: "Player"},
	)
	fmt.Printf("创建 Player (ID: %d)\n", player)

	enemy := world.CreateEntity(
		Position{X: 10, Y: 5},
		Velocity{DX: -2, DY: 0},
		Health{Current: 30, Max: 30},
		Name{Value: "Enemy"},
	)
	fmt.Printf("创建 Enemy (ID: %d)\n", enemy)

	// 4. 主循环
	fmt.Println("\n--- 开始模拟 ---\n")
	fps := float64(2)
	dt := time.Duration(float64(time.Second) / fps)

	for i := 1; i <= 10; i++ {
		fmt.Printf("== 第 %d 帧 ==\n", i)

		// 模拟：第3帧对 Enemy 造成伤害
		if i == 3 {
			damageSys.QueueDamage(enemy, 15)
		}
		// 模拟：第5帧再造成一次伤害
		if i == 5 {
			damageSys.QueueDamage(enemy, 20)
		}

		world.Step(dt)

		// 打印实体状态
		printEntity(world, player)
		printEntity(world, enemy)
		fmt.Println()
	}

	// 5. 统计信息
	stats := world.Stats()
	fmt.Printf("=== 统计 ===\n")
	fmt.Printf("总实体数: %d\n", stats.TotalEntities)
	fmt.Printf("存活实体: %d\n", stats.AliveEntities)
	fmt.Printf("系统数量: %d\n", stats.TotalSystems)
	fmt.Printf("组件类型: %d\n", stats.ComponentTypes)
	fmt.Printf("组件总数: %d\n", stats.ComponentCount)

	// 6. 清理
	world.Destroy()
}

func printEntity(world ecs.World, entityID ecs.EntityID) {
	if !world.EntityManager().IsAlive(entityID) {
		return
	}
	name := getEntityName(world, entityID)
	comp, _ := world.ComponentManager().GetComponent(entityID, reflect.TypeOf(Position{}))
	pos := comp.(Position)
	fmt.Printf("  %s -> 位置: (%.1f, %.1f)\n", name, pos.X, pos.Y)
}
