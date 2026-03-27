package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
)

// TestUser 测试用户模型
type TestUser struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"size:100;not null"`
	Email     string `gorm:"size:100;uniqueIndex;not null"`
	Age       int    `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// testConfig 创建测试配置
func testConfig() *Config {
	return &Config{
		Driver:                    "sqlite",
		Database:                  ":memory:",
		LogLevel:                  "info",
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: false,
		ParameterizedQueries:      true,
		Colorful:                  false,
		MaxIdleConns:              10,
		MaxOpenConns:              100,
		ConnMaxLifetime:           time.Hour,
		ConnMaxIdleTime:           10 * time.Minute,
		PrepareStmt:               true,
	}
}

// testDatabase 创建测试数据库
func testDatabase(t *testing.T) *Database {
	config := testConfig()
	db, err := New(config)
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	return db
}

func closeTestDatabase(t *testing.T, db *Database) {
	t.Helper()

	if err := db.Close(); err != nil {
		t.Fatalf("关闭测试数据库失败: %v", err)
	}
}

func resetDefaultClientForTest(t *testing.T) {
	t.Helper()

	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()

	if defaultClient != nil {
		if err := defaultClient.Close(); err != nil {
			t.Fatalf("关闭默认客户端失败: %v", err)
		}
	}

	defaultClient = nil
}

// TestConfig_Validate 测试配置验证
func TestConfig_Validate(t *testing.T) {
	t.Run("有效配置", func(t *testing.T) {
		config := &Config{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Password: "password",
			Database: "test_db",
		}

		config.SetDefaults()
		err := config.Validate()
		if err != nil {
			t.Errorf("有效配置验证失败: %v", err)
		}
	})

	t.Run("无效配置 - 缺少驱动", func(t *testing.T) {
		config := &Config{}

		err := config.Validate()
		if err == nil {
			t.Error("期望配置验证失败，但没有错误")
		}
		if !strings.Contains(err.Error(), "数据库驱动不能为空") {
			t.Errorf("错误消息不匹配: %v", err)
		}
	})

	t.Run("无效配置 - 不支持的驱动", func(t *testing.T) {
		config := &Config{
			Driver: "oracle",
		}

		err := config.Validate()
		if err == nil {
			t.Error("期望配置验证失败，但没有错误")
		}
		if !strings.Contains(err.Error(), "不支持的数据库驱动") {
			t.Errorf("错误消息不匹配: %v", err)
		}
	})

	t.Run("无效配置 - 缺少主机", func(t *testing.T) {
		config := &Config{
			Driver: "mysql",
		}

		err := config.Validate()
		if err == nil {
			t.Error("期望配置验证失败，但没有错误")
		}
		if !strings.Contains(err.Error(), "数据库主机不能为空") {
			t.Errorf("错误消息不匹配: %v", err)
		}
	})

	t.Run("SQLite配置", func(t *testing.T) {
		config := &Config{
			Driver:   "sqlite",
			Database: "test.db",
		}

		config.SetDefaults()
		err := config.Validate()
		if err != nil {
			t.Errorf("SQLite配置验证失败: %v", err)
		}
	})
}

// TestNew 测试创建数据库
func TestNew(t *testing.T) {
	t.Run("成功创建", func(t *testing.T) {
		config := testConfig()
		db, err := New(config)
		if err != nil {
			t.Fatalf("创建数据库失败: %v", err)
		}
		defer closeTestDatabase(t, db)

		// 测试连接
		err = db.Ping()
		if err != nil {
			t.Errorf("数据库连接测试失败: %v", err)
		}
	})

	t.Run("无效配置", func(t *testing.T) {
		config := &Config{
			Driver: "invalid",
		}

		_, err := New(config)
		if err == nil {
			t.Error("期望创建失败，但没有错误")
		}
	})
}

func TestConfigureAndGetClient(t *testing.T) {
	resetDefaultClientForTest(t)
	t.Cleanup(func() {
		resetDefaultClientForTest(t)
	})

	client, err := Configure(testConfig())
	if err != nil {
		t.Fatalf("Configure() 失败: %v", err)
	}
	if client == nil {
		t.Fatal("Configure() 返回了空客户端")
	}

	got := GetClient()
	if got == nil {
		t.Fatal("GetClient() 返回了空客户端")
	}
	if got != client {
		t.Fatal("GetClient() 没有返回已配置客户端")
	}
}

func TestResolveClient(t *testing.T) {
	resetDefaultClientForTest(t)
	t.Cleanup(func() {
		resetDefaultClientForTest(t)
	})

	override := testDatabase(t)
	defer closeTestDatabase(t, override)

	tests := []struct {
		name      string
		setup     func(t *testing.T)
		overrides []*Database
		wantErr   error
	}{
		{
			name: "优先返回显式覆盖客户端",
			setup: func(t *testing.T) {
				_, err := Configure(testConfig())
				if err != nil {
					t.Fatalf("Configure() 失败: %v", err)
				}
			},
			overrides: []*Database{override},
		},
		{
			name: "未覆盖时返回默认客户端",
			setup: func(t *testing.T) {
				_, err := Configure(testConfig())
				if err != nil {
					t.Fatalf("Configure() 失败: %v", err)
				}
			},
		},
		{
			name: "未配置默认客户端时返回错误",
			setup: func(t *testing.T) {
				resetDefaultClientForTest(t)
			},
			wantErr: ErrMissingClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDefaultClientForTest(t)
			tt.setup(t)

			got, err := resolveClient(tt.overrides...)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("resolveClient() 期望失败，但返回了 nil")
				}
				if err != tt.wantErr {
					t.Fatalf("resolveClient() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveClient() 失败: %v", err)
			}
			if got == nil {
				t.Fatal("resolveClient() 返回了空客户端")
			}

			if len(tt.overrides) > 0 && got != tt.overrides[0] {
				t.Fatal("resolveClient() 没有优先返回显式覆盖客户端")
			}
			if len(tt.overrides) == 0 && got != GetClient() {
				t.Fatal("resolveClient() 没有返回默认客户端")
			}
		})
	}
}

func TestPackageLevelHelpers(t *testing.T) {
	resetDefaultClientForTest(t)
	t.Cleanup(func() {
		resetDefaultClientForTest(t)
	})

	t.Run("Ping在未配置默认客户端时返回错误", func(t *testing.T) {
		err := Ping()
		if err != ErrMissingClient {
			t.Fatalf("Ping() error = %v, want %v", err, ErrMissingClient)
		}
	})

	t.Run("Ping使用默认客户端", func(t *testing.T) {
		_, err := Configure(testConfig())
		if err != nil {
			t.Fatalf("Configure() 失败: %v", err)
		}

		if err := Ping(); err != nil {
			t.Fatalf("Ping() 失败: %v", err)
		}
	})

	t.Run("Exec和Query支持显式覆盖客户端", func(t *testing.T) {
		override := testDatabase(t)
		defer closeTestDatabase(t, override)

		if err := override.AutoMigrate(&TestUser{}); err != nil {
			t.Fatalf("AutoMigrate() 失败: %v", err)
		}

		ctx := context.Background()
		err := Exec(
			ctx,
			"INSERT INTO test_users(name, email, age, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			[]interface{}{"包级用户", "pkg@example.com", 18, time.Now(), time.Now()},
			override,
		)
		if err != nil {
			t.Fatalf("Exec() 失败: %v", err)
		}

		var user TestUser
		err = Query(
			ctx,
			&user,
			"SELECT * FROM test_users WHERE email = ? LIMIT 1",
			[]interface{}{"pkg@example.com"},
			override,
		)
		if err != nil {
			t.Fatalf("Query() 失败: %v", err)
		}
		if user.Email != "pkg@example.com" {
			t.Fatalf("Query() 返回了错误记录: %+v", user)
		}
	})
}

// TestDatabase_GetDB 测试获取GORM实例
func TestDatabase_GetDB(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	gormDB := db.GetDB()
	if gormDB == nil {
		t.Error("GORM实例不能为空")
	}
}

// TestDatabase_Ping 测试数据库连接
func TestDatabase_Ping(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	err := db.Ping()
	if err != nil {
		t.Errorf("数据库连接测试失败: %v", err)
	}
}

// TestDatabase_Stats 测试连接池统计
func TestDatabase_Stats(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	stats := db.Stats()
	if stats.OpenConnections < 0 {
		t.Error("打开连接数不能为负数")
	}
}

// TestDatabase_AutoMigrate 测试自动迁移
func TestDatabase_AutoMigrate(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	err := db.AutoMigrate(&TestUser{})
	if err != nil {
		t.Errorf("自动迁移失败: %v", err)
	}
}

// TestDatabase_CRUD 测试CRUD操作
func TestDatabase_CRUD(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	// 自动迁移
	err := db.AutoMigrate(&TestUser{})
	if err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	gormDB := db.GetDB()
	ctx := context.Background()

	t.Run("创建", func(t *testing.T) {
		user := &TestUser{
			Name:  "张三",
			Email: "zhangsan@example.com",
			Age:   25,
		}

		result := gormDB.WithContext(ctx).Create(user)
		if result.Error != nil {
			t.Errorf("创建用户失败: %v", result.Error)
		}
		if user.ID == 0 {
			t.Error("用户ID应该被自动设置")
		}
	})

	t.Run("查询", func(t *testing.T) {
		var user TestUser
		result := gormDB.WithContext(ctx).Where("email = ?", "zhangsan@example.com").First(&user)
		if result.Error != nil {
			t.Errorf("查询用户失败: %v", result.Error)
		}
		if user.Name != "张三" {
			t.Errorf("用户名不匹配，期望: 张三, 实际: %s", user.Name)
		}
	})

	t.Run("更新", func(t *testing.T) {
		result := gormDB.WithContext(ctx).Model(&TestUser{}).Where("email = ?", "zhangsan@example.com").Update("age", 26)
		if result.Error != nil {
			t.Errorf("更新用户失败: %v", result.Error)
		}
		if result.RowsAffected != 1 {
			t.Errorf("期望更新1行，实际更新了%d行", result.RowsAffected)
		}
	})

	t.Run("删除", func(t *testing.T) {
		result := gormDB.WithContext(ctx).Where("email = ?", "zhangsan@example.com").Delete(&TestUser{})
		if result.Error != nil {
			t.Errorf("删除用户失败: %v", result.Error)
		}
		if result.RowsAffected != 1 {
			t.Errorf("期望删除1行，实际删除了%d行", result.RowsAffected)
		}
	})
}

// TestDatabase_Transaction 测试事务操作
func TestDatabase_Transaction(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	// 自动迁移
	err := db.AutoMigrate(&TestUser{})
	if err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	gormDB := db.GetDB()
	ctx := context.Background()

	t.Run("成功事务", func(t *testing.T) {
		err := gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// 创建用户1
			user1 := &TestUser{
				Name:  "用户1",
				Email: "user1@example.com",
				Age:   20,
			}
			if err := tx.Create(user1).Error; err != nil {
				return err
			}

			// 创建用户2
			user2 := &TestUser{
				Name:  "用户2",
				Email: "user2@example.com",
				Age:   25,
			}
			if err := tx.Create(user2).Error; err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			t.Errorf("事务执行失败: %v", err)
		}

		// 验证两个用户都被创建
		var count int64
		gormDB.WithContext(ctx).Model(&TestUser{}).Count(&count)
		if count != 2 {
			t.Errorf("期望2个用户，实际%d个", count)
		}
	})

	t.Run("失败事务回滚", func(t *testing.T) {
		// 先清理数据
		gormDB.WithContext(ctx).Where("email IN ?", []string{"user1@example.com", "user2@example.com"}).Delete(&TestUser{})

		err := gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// 创建用户1
			user1 := &TestUser{
				Name:  "用户1",
				Email: "user1@example.com",
				Age:   20,
			}
			if err := tx.Create(user1).Error; err != nil {
				return err
			}

			// 故意创建一个会失败的记录（重复邮箱）
			user2 := &TestUser{
				Name:  "用户2",
				Email: "user1@example.com", // 重复邮箱
				Age:   25,
			}
			if err := tx.Create(user2).Error; err != nil {
				return err
			}

			return nil
		})

		if err == nil {
			t.Error("期望事务失败，但没有错误")
		}

		// 验证没有用户被创建（回滚成功）
		var count int64
		gormDB.WithContext(ctx).Model(&TestUser{}).Count(&count)
		if count != 0 {
			t.Errorf("期望0个用户（回滚后），实际%d个", count)
		}
	})
}

func TestDatabase_Tx(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	if err := db.AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	ctx := context.Background()
	expectedErr := errors.New("transaction callback failed")

	err := db.Tx(ctx, func(tx *gorm.DB) error {
		user := &TestUser{
			Name:  "事务用户",
			Email: "tx@example.com",
			Age:   20,
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Tx() error = %v, want %v", err, expectedErr)
	}

	var count int64
	if err := db.GetDB().WithContext(ctx).Model(&TestUser{}).Where("email = ?", "tx@example.com").Count(&count).Error; err != nil {
		t.Fatalf("统计事务数据失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("期望事务回滚后数据为0条，实际为%d条", count)
	}
}

// TestDatabase_ErrorHandling 测试错误处理
func TestDatabase_ErrorHandling(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	// 自动迁移
	err := db.AutoMigrate(&TestUser{})
	if err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	gormDB := db.GetDB()
	ctx := context.Background()

	t.Run("查询不存在的记录", func(t *testing.T) {
		var user TestUser
		result := gormDB.WithContext(ctx).Where("email = ?", "nonexistent@example.com").First(&user)
		if result.Error == nil {
			t.Error("期望查询失败，但没有错误")
		}
		if !strings.Contains(result.Error.Error(), "record not found") {
			t.Errorf("错误消息不匹配: %v", result.Error)
		}
	})

	t.Run("创建重复记录", func(t *testing.T) {
		// 先创建一个用户
		user1 := &TestUser{
			Name:  "测试用户",
			Email: "test@example.com",
			Age:   25,
		}
		if err := gormDB.WithContext(ctx).Create(user1).Error; err != nil {
			t.Fatalf("创建第一个用户失败: %v", err)
		}

		// 尝试创建相同邮箱的用户
		user2 := &TestUser{
			Name:  "测试用户2",
			Email: "test@example.com", // 重复邮箱
			Age:   30,
		}
		result := gormDB.WithContext(ctx).Create(user2)
		if result.Error == nil {
			t.Error("期望创建失败，但没有错误")
		}
	})
}

// TestDatabase_ConcurrentAccess 测试并发访问
func TestDatabase_ConcurrentAccess(t *testing.T) {
	db := testDatabase(t)
	defer closeTestDatabase(t, db)

	// 自动迁移
	err := db.AutoMigrate(&TestUser{})
	if err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	gormDB := db.GetDB()
	ctx := context.Background()

	// 先创建一个测试用户确保表已创建并验证表存在
	testUser := &TestUser{
		Name:  "测试用户",
		Email: "test@example.com",
		Age:   25,
	}
	if err := gormDB.WithContext(ctx).Create(testUser).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 验证表确实存在并且可以正常操作
	var count int64
	if err := gormDB.WithContext(ctx).Model(&TestUser{}).Count(&count).Error; err != nil {
		t.Fatalf("验证表存在失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("期望1个测试用户，实际%d个", count)
	}

	// 减少并发数量，SQLite内存数据库对并发支持有限
	const numGoroutines = 5
	const numOperations = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 使用互斥锁来确保数据库操作的原子性
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// 创建用户
				user := &TestUser{
					Name:  fmt.Sprintf("用户%d-%d", id, j),
					Email: fmt.Sprintf("user%d-%d@example.com", id, j),
					Age:   20 + (id+j)%30,
				}

				mu.Lock()
				err := gormDB.WithContext(ctx).Create(user).Error
				mu.Unlock()

				if err != nil {
					t.Errorf("并发创建用户失败: %v", err)
					return
				}

				// 查询用户
				var foundUser TestUser
				mu.Lock()
				err = gormDB.WithContext(ctx).Where("email = ?", user.Email).First(&foundUser).Error
				mu.Unlock()

				if err != nil {
					t.Errorf("并发查询用户失败: %v", err)
					return
				}

				// 更新用户
				mu.Lock()
				err = gormDB.WithContext(ctx).Model(&foundUser).Update("age", foundUser.Age+1).Error
				mu.Unlock()

				if err != nil {
					t.Errorf("并发更新用户失败: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证所有操作都成功
	mu.Lock()
	err = gormDB.WithContext(ctx).Model(&TestUser{}).Count(&count).Error
	mu.Unlock()

	if err != nil {
		t.Fatalf("最终统计失败: %v", err)
	}
	expectedCount := int64(numGoroutines*numOperations + 1) // +1 for the test user
	if count != expectedCount {
		t.Errorf("期望%d个用户，实际%d个", expectedCount, count)
	}
}
