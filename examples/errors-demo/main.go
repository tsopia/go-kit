package main

import (
	stderrors "errors"
	"fmt"

	kiterr "github.com/tsopia/go-kit/errors"
)

func main() {
	fmt.Println("=== errors 包新 API 示例 ===")

	// 1) 显式注册业务错误（Register）
	errBiz := kiterr.Register(6100, "BIZ_ERROR").WithHTTP(400)
	errOrder := kiterr.Register(6101, "ORDER_ERROR").WithHTTP(409)
	errOrder.Class = errBiz
	fmt.Printf("1. Register: %s(code=%d,http=%d)\n", errOrder.Name, errOrder.Code, errOrder.HTTP)

	// 2) 自动分配错误定义（NewDefinition）
	autoA := kiterr.NewDefinition("ORDER_EXPIRED")
	autoB := kiterr.NewDefinition("ORDER_EXPIRED")
	autoA.HTTP = 410
	autoA.Class = errOrder
	fmt.Printf("2. NewDefinition: name=%s, code=%d, 同名复用=%v\n", autoA.Name, autoA.Code, autoA == autoB)

	// 3) 创建错误（New/Newf/Wrap/Wrapf）
	errNew := autoA.New("订单已过期")
	errNewf := autoA.Newf("订单 %s 已过期", "o-1001")
	cause := fmt.Errorf("db timeout")
	errWrap := autoA.Wrap(cause, "查询订单失败")
	errWrapf := autoA.Wrapf(cause, "查询订单失败, order=%s", "o-1001")

	fmt.Printf("3. New: %v\n", errNew)
	fmt.Printf("3. Newf: %v\n", errNewf)
	fmt.Printf("3. Wrap: %v\n", errWrap)
	fmt.Printf("3. Wrapf: %v\n", errWrapf)

	// 4) 判断错误（errors.Is + 层级）
	fmt.Printf("4. errors.Is(errWrapf, autoA) = %v\n", stderrors.Is(errWrapf, autoA))
	fmt.Printf("4. errors.Is(errWrapf, errOrder) = %v\n", stderrors.Is(errWrapf, errOrder))
	fmt.Printf("4. errors.Is(errWrapf, errBiz) = %v\n", stderrors.Is(errWrapf, errBiz))
	fmt.Printf("4. errors.Is(errWrapf, cause) = %v\n", stderrors.Is(errWrapf, cause))

	// 5) 获取错误信息（Code/Name/HTTPCode）
	fmt.Printf("5. Code(errWrapf) = %d\n", kiterr.Code(errWrapf))
	fmt.Printf("5. Name(errWrapf) = %s\n", kiterr.Name(errWrapf))
	fmt.Printf("5. HTTPCode(errWrapf) = %d\n", kiterr.HTTPCode(errWrapf))

	// 6) 打印错误信息
	fmt.Printf("6. 打印错误: %v\n", errWrapf)
}
