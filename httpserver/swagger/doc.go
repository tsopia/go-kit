// Package swagger 提供 Swagger UI 的 Gin 路由挂载能力。
//
// 该包只负责运行时挂载 Swagger UI，不负责生成 OpenAPI 文档。
// 业务项目需要先使用 swaggo 生成文档代码，并在主程序中显式导入生成包。
//
// 推荐把 Swagger 注册在公共路由组上：
//
//	public := srv.Group("")
//	protected := srv.Group("/api/v1")
//	protected.Use(AuthMiddleware())
//
//	swagger.Register(public, swagger.Config{})
package swagger
