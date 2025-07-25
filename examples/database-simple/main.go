package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tsopia/go-kit/pkg/database"
)

// User 用户模型
type User struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"size:100;not null"`
	Email     string `gorm:"size:100;uniqueIndex;not null"`
	Age       int    `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	// 示例1: 使用默认值配置
	fmt.Println("=== 示例1: 使用默认值配置 ===")
	config1 := &database.Config{
		Driver:   "sqlite",
		Database: ":memory:",
		LogLevel: "info",
		// 其他配置将使用默认值
	}

	db1, err := database.New(config1)
	if err != nil {
		log.Fatalf("创建数据库失败: %v", err)
	}
	defer db1.Close()

	fmt.Println("数据库连接成功，使用默认配置")

	// 示例2: 明确设置配置值
	fmt.Println("\n=== 示例2: 明确设置配置值 ===")

	config2 := &database.Config{
		Driver:                    "sqlite",
		Database:                  ":memory:",
		LogLevel:                  "info",
		IgnoreRecordNotFoundError: false, // 明确设置为 false
		ParameterizedQueries:      true,  // 明确设置为 true
		MaxIdleConns:              5,     // 自定义连接池大小
		MaxOpenConns:              50,
		ConnMaxLifetime:           30 * time.Minute,
		SlowThreshold:             100 * time.Millisecond,
	}

	db2, err := database.New(config2)
	if err != nil {
		log.Fatalf("创建数据库失败: %v", err)
	}
	defer db2.Close()

	fmt.Println("数据库连接成功，使用明确设置的配置")

	// 示例3: 混合设置
	fmt.Println("\n=== 示例3: 混合设置 ===")
	config3 := &database.Config{
		Driver:                    "sqlite",
		Database:                  ":memory:",
		LogLevel:                  "info",
		IgnoreRecordNotFoundError: true, // 明确设置为 true
		// 其他配置使用默认值
	}

	db3, err := database.New(config3)
	if err != nil {
		log.Fatalf("创建数据库失败: %v", err)
	}
	defer db3.Close()

	fmt.Println("数据库连接成功，混合配置")

	// 示例4: 健康检查和统计信息
	fmt.Println("\n=== 示例4: 健康检查和统计信息 ===")

	// 执行健康检查
	if err := db1.HealthCheck(); err != nil {
		log.Printf("健康检查失败: %v", err)
	} else {
		fmt.Println("健康检查通过")
	}

	// 获取连接池统计信息
	stats := db1.Stats()
	fmt.Printf("连接池统计: 打开连接数=%d, 空闲连接数=%d\n",
		stats.OpenConnections, stats.IdleConnections)

	// 示例5: 数据库操作测试
	fmt.Println("\n=== 示例5: 数据库操作测试 ===")

	// 自动迁移
	if err := db1.AutoMigrate(&User{}); err != nil {
		log.Fatalf("自动迁移失败: %v", err)
	}
	fmt.Println("自动迁移成功")

	// 创建用户
	user := User{
		Name:  "张三",
		Email: "zhangsan@example.com",
		Age:   25,
	}

	gormDB := db1.GetDB()
	if err := gormDB.Create(&user).Error; err != nil {
		log.Fatalf("创建用户失败: %v", err)
	}
	fmt.Printf("用户创建成功，ID: %d\n", user.ID)

	// 查询用户
	var retrievedUser User
	if err := gormDB.First(&retrievedUser, user.ID).Error; err != nil {
		log.Fatalf("查询用户失败: %v", err)
	}
	fmt.Printf("查询到用户: %s, 邮箱: %s, 年龄: %d\n",
		retrievedUser.Name, retrievedUser.Email, retrievedUser.Age)

	// 更新用户
	if err := gormDB.Model(&retrievedUser).Update("Age", 26).Error; err != nil {
		log.Fatalf("更新用户失败: %v", err)
	}
	fmt.Println("用户更新成功")

	// 删除用户
	if err := gormDB.Delete(&retrievedUser).Error; err != nil {
		log.Fatalf("删除用户失败: %v", err)
	}
	fmt.Println("用户删除成功")

	// 示例6: 配置验证
	fmt.Println("\n=== 示例6: 配置验证 ===")

	// 测试无效配置
	invalidConfig := &database.Config{
		Driver:   "unsupported",
		Database: "",
	}

	_, err = database.New(invalidConfig)
	if err != nil {
		fmt.Printf("预期的配置验证错误: %v\n", err)
	}

	// 测试端口范围验证
	invalidPortConfig := &database.Config{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     70000, // 超出范围
		Username: "user",
		Database: "test",
	}

	_, err = database.New(invalidPortConfig)
	if err != nil {
		fmt.Printf("预期的端口验证错误: %v\n", err)
	}

	fmt.Println("\n=== 所有示例完成 ===")
}
