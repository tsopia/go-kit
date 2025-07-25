package main

import (
	"fmt"
	"log"

	"github.com/tsopia/go-kit/pkg/errors"
)

func main() {
	fmt.Println("=== Go-Kit Errors Package Demo ===\n")

	// 1. 基本错误创建
	demoBasicErrors()

	// 2. 错误包装
	demoErrorWrapping()

	// 3. 错误检查
	demoErrorChecking()

	// 4. 链式调用
	demoFluentInterface()

	// 5. 格式化错误
	demoFormattedErrors()

	// 6. 工具函数
	demoUtilityFunctions()

	// 7. 便利函数
	demoConvenienceFunctions()

	// 8. 实际应用场景
	demoRealWorldScenarios()
}

func demoBasicErrors() {
	fmt.Println("1. 基本错误创建")
	fmt.Println("----------------")

	// 使用默认消息
	err1 := errors.New(errors.CodeInvalidParam)
	fmt.Printf("默认消息错误: %v\n", err1)

	// 使用自定义消息
	err2 := errors.New(errors.CodeNotFound, "用户不存在")
	fmt.Printf("自定义消息错误: %v\n", err2)

	// 带详细信息的错误
	err3 := errors.NewWithDetails(errors.CodeDatabaseError, "数据库连接失败", "连接超时")
	fmt.Printf("带详细信息错误: %v\n", err3)

	fmt.Println()
}

func demoErrorWrapping() {
	fmt.Println("2. 错误包装")
	fmt.Println("------------")

	// 包装标准库错误
	originalErr := fmt.Errorf("文件读取失败")
	wrappedErr := errors.Wrap(originalErr, errors.CodeNotFound, "资源不存在")
	fmt.Printf("包装错误: %v\n", wrappedErr)

	// 包装并添加详细信息
	wrappedWithDetails := errors.WrapWithDetails(
		originalErr,
		errors.CodeDatabaseError,
		"数据库操作失败",
		"SQL执行超时",
	)
	fmt.Printf("包装带详细信息: %v\n", wrappedWithDetails)

	fmt.Println()
}

func demoErrorChecking() {
	fmt.Println("3. 错误检查")
	fmt.Println("------------")

	// 创建不同类型的错误
	err1 := errors.New(errors.CodeInvalidParam, "参数无效")
	err2 := errors.New(errors.CodeNotFound, "资源不存在")
	err3 := errors.New(errors.CodeDatabaseError, "数据库错误")

	// 检查错误类型
	fmt.Printf("err1 是参数错误: %v\n", errors.Is(err1, errors.CodeInvalidParam))
	fmt.Printf("err1 是未找到错误: %v\n", errors.Is(err1, errors.CodeNotFound))

	// 使用便利函数检查
	fmt.Printf("err2 是未找到错误: %v\n", errors.IsNotFound(err2))
	fmt.Printf("err3 是数据库错误: %v\n", errors.IsDatabaseError(err3))

	// 检查错误码
	code := errors.GetCode(err1)
	fmt.Printf("err1 的错误码: %s (代码: %d)\n", code.Name, code.Code)

	fmt.Println()
}

func demoFluentInterface() {
	fmt.Println("4. 链式调用")
	fmt.Println("------------")

	// 创建错误并添加上下文和堆栈
	err := errors.New(errors.CodeUnauthorized, "访问被拒绝").
		WithContext("user_id", "12345").
		WithContext("ip", "192.168.1.100").
		WithDetails("用户权限不足").
		WithStack()

	fmt.Printf("链式调用错误: %v\n", err)
	fmt.Printf("错误上下文: %v\n", err.Context)
	fmt.Printf("错误堆栈: %s\n", err.Stack[:100]) // 只显示前100个字符

	fmt.Println()
}

func demoFormattedErrors() {
	fmt.Println("5. 格式化错误")
	fmt.Println("--------------")

	// 使用格式化创建错误
	userID := "john_doe"
	err1 := errors.Newf(errors.CodeUserNotFound, "用户 %s 不存在", userID)
	fmt.Printf("格式化错误: %v\n", err1)

	// 使用格式化包装错误
	originalErr := fmt.Errorf("网络连接失败")
	err2 := errors.Wrapf(originalErr, errors.CodeNetworkError, "无法连接到服务器 %s", "api.example.com")
	fmt.Printf("格式化包装错误: %v\n", err2)

	fmt.Println()
}

func demoUtilityFunctions() {
	fmt.Println("6. 工具函数")
	fmt.Println("------------")

	// 字符串转错误码
	code1 := errors.StringToCode("INVALID_PARAM")
	fmt.Printf("字符串转错误码: %v\n", code1)

	code2, found := errors.StringToCodeWithFound("NOT_FOUND")
	fmt.Printf("字符串转错误码(带found): %v, found: %v\n", code2, found)

	// 创建自定义错误码
	customCode := errors.NewErrorCode(5001, "CUSTOM_ERROR", "自定义错误")
	fmt.Printf("自定义错误码: %v\n", customCode)

	fmt.Println()
}

func demoConvenienceFunctions() {
	fmt.Println("7. 便利函数")
	fmt.Println("------------")

	// 使用便利函数快速创建错误
	err1 := errors.InvalidParam("参数不能为空")
	err2 := errors.NotFound("用户不存在")
	err3 := errors.Internal("内部服务器错误")

	fmt.Printf("便利函数错误1: %v\n", err1)
	fmt.Printf("便利函数错误2: %v\n", err2)
	fmt.Printf("便利函数错误3: %v\n", err3)

	fmt.Println()
}

func demoRealWorldScenarios() {
	fmt.Println("8. 实际应用场景")
	fmt.Println("----------------")

	// 模拟用户服务
	demoUserService()

	// 模拟数据库操作
	demoDatabaseOperations()

	// 模拟API调用
	demoAPICalls()
}

func demoUserService() {
	fmt.Println("\n用户服务示例:")

	// 模拟用户查找
	userID := "user123"
	if userID == "" {
		err := errors.InvalidParam("用户ID不能为空")
		log.Printf("用户服务错误: %v", err)
		return
	}

	// 模拟用户不存在
	if userID == "user123" {
		err := errors.NotFound("用户不存在").
			WithContext("user_id", userID).
			WithContext("request_id", "req-456")
		log.Printf("用户服务错误: %v", err)
		return
	}

	// 模拟权限检查
	if userID == "admin" {
		err := errors.New(errors.CodeUnauthorized, "权限不足").
			WithDetails("需要管理员权限")
		log.Printf("用户服务错误: %v", err)
		return
	}
}

func demoDatabaseOperations() {
	fmt.Println("\n数据库操作示例:")

	// 模拟数据库连接失败
	dbErr := errors.New(errors.CodeDatabaseError, "数据库连接失败").
		WithContext("db_host", "localhost").
		WithContext("db_port", 5432).
		WithDetails("连接超时")
	log.Printf("数据库错误: %v", dbErr)

	// 模拟记录不存在
	recordErr := errors.New(errors.CodeRecordNotFound, "记录不存在").
		WithContext("table", "users").
		WithContext("id", 123)
	log.Printf("数据库错误: %v", recordErr)
}

func demoAPICalls() {
	fmt.Println("\nAPI调用示例:")

	// 模拟外部服务调用失败
	apiErr := errors.New(errors.CodeExternalServiceError, "外部服务调用失败").
		WithContext("service", "payment-gateway").
		WithContext("endpoint", "/api/v1/payments").
		WithDetails("HTTP 500 错误")
	log.Printf("API错误: %v", apiErr)

	// 模拟网络超时
	timeoutErr := errors.New(errors.CodeTimeoutError, "请求超时").
		WithContext("timeout", "30s").
		WithContext("retry_count", 3)
	log.Printf("API错误: %v", timeoutErr)
}
