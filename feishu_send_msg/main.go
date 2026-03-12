package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/larksuite/oapi-sdk-go/v3"
	// larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// Config 配置结构体，对应config.json
type Config struct {
	AppID         string `json:"app_id"`          // 必填
	AppSecret     string `json:"app_secret"`      // 必填
	ReceiveIdType string `json:"receive_id_type"` // 可选，默认email
	ReceiveId     string `json:"receive_id"`      // 必填
	Content       string `json:"content"`         // 必填
	Uuid          string `json:"uuid"`            // 可选
	ExitDelay     int    `json:"exit_delay"`      // 新增：退出延时（秒），0=直接退出
}

func main() {
	// 1. 定义命令行参数（新增-exit-delay参数）
	var (
		appID         = flag.String("app-id", "", "飞书应用ID")
		appSecret     = flag.String("app-secret", "", "飞书应用密钥")
		receiveIdType = flag.String("receive-id-type", "", "接收ID类型")
		receiveId     = flag.String("receive-id", "", "接收ID")
		content       = flag.String("content", "", "发送的文本内容")
		uuid          = flag.String("uuid", "", "消息UUID")
		exitDelay     = flag.Int("exit-delay", 0, "运行完成后退出延时（秒），0=直接退出") // 新增参数
	)
	flag.Parse()

	// 2. 初始化最终配置
	var finalConfig Config

	// 3. 判断是否传入了命令行核心参数
	hasCLIParams := *appID != "" || *appSecret != "" || *receiveId != "" || *content != ""
	if hasCLIParams {
		// 命令行模式：校验必填参数必须全传
		if *appID == "" || *appSecret == "" || *receiveId == "" || *content == "" {
			fmt.Println("错误：命令行模式下必须传入所有核心参数！")
			fmt.Println("必填参数：-app-id、-app-secret、-receive-id、-content")
			fmt.Println("可选参数：-receive-id-type（默认email）、-uuid、-exit-delay（默认0）")
			os.Exit(1)
		}

		// 赋值命令行参数到最终配置
		finalConfig = Config{
			AppID:         *appID,
			AppSecret:     *appSecret,
			ReceiveIdType: *receiveIdType,
			ReceiveId:     *receiveId,
			Content:       *content,
			Uuid:          *uuid,
			ExitDelay:     *exitDelay, // 赋值退出延时参数
		}
		// 命令行模式下，receiveIdType默认值
		if finalConfig.ReceiveIdType == "" {
			finalConfig.ReceiveIdType = "email"
		}
		fmt.Println("===== 使用命令行参数模式 =====")
	} else {
		// 配置文件模式：固定读取当前目录的config.json
		configFile := "config.json"
		data, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Printf("错误：读取当前目录的%s失败 - %v\n", configFile, err)
			fmt.Println("提示：要么确保当前目录有合法的config.json，要么通过命令行传入参数运行")
			os.Exit(1)
		}

		// 解析配置文件
		if err := json.Unmarshal(data, &finalConfig); err != nil {
			fmt.Printf("错误：解析%s失败 - %v\n", configFile, err)
			os.Exit(1)
		}

		// 校验配置文件必填参数
		if finalConfig.AppID == "" || finalConfig.AppSecret == "" || finalConfig.ReceiveId == "" || finalConfig.Content == "" {
			fmt.Println("错误：config.json中缺少必填字段（app_id/app_secret/receive_id/content）")
			os.Exit(1)
		}

		// 配置文件模式下，设置默认值
		if finalConfig.ReceiveIdType == "" {
			finalConfig.ReceiveIdType = "email"
		}
		if finalConfig.ExitDelay < 0 { // 防止配置文件中设置负数
			finalConfig.ExitDelay = 0
		}
		fmt.Println("===== 使用配置文件模式（config.json） =====")
	}

	// 4. 打印最终生效的所有参数（新增核心功能）
	fmt.Println("===== 最终生效参数 =====")
	fmt.Printf("AppID:         %s\n", finalConfig.AppID)
	fmt.Printf("AppSecret:     %s（已脱敏）\n", maskString(finalConfig.AppSecret)) // 脱敏显示密钥
	fmt.Printf("ReceiveIdType: %s\n", finalConfig.ReceiveIdType)
	fmt.Printf("ReceiveId:     %s\n", finalConfig.ReceiveId)
	fmt.Printf("原始消息内容:  %s\n", finalConfig.Content)
	fmt.Printf("UUID:          %s\n", finalConfig.Uuid)
	fmt.Printf("退出延时:      %d 秒\n", finalConfig.ExitDelay)
	fmt.Println("======================")

	// 5. 消息内容自动拼接当前时间（新增核心功能）
	now := time.Now().Format("2006-01-02 15:04:05") // Go的时间格式化固定模板
	contentWithTime := fmt.Sprintf("[%s] %s", now, finalConfig.Content)
	fmt.Printf("最终发送消息:  %s\n", contentWithTime)

	// 6. 创建飞书客户端并发送消息
	client := lark.NewClient(finalConfig.AppID, finalConfig.AppSecret)
	contentJSON := fmt.Sprintf(`{"text":"%s"}`, contentWithTime)

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(finalConfig.ReceiveIdType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(finalConfig.ReceiveId).
			MsgType("text").
			Content(contentJSON).
			Uuid(finalConfig.Uuid).
			Build()).
		Build()

	// 发起请求
	resp, err := client.Im.V1.Message.Create(context.Background(), req)
	if err != nil {
		fmt.Printf("发送消息失败: %v\n", err)
		// 即使发送失败，也按延时参数退出
		handleExitDelay(finalConfig.ExitDelay)
		os.Exit(1)
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Printf("请求失败 - logId: %s, 错误码: %d, 错误信息: %s\n",
			resp.RequestId(), resp.CodeError.Code, resp.CodeError.Msg)
		handleExitDelay(finalConfig.ExitDelay)
		os.Exit(1)
	}

	// 成功提示
	fmt.Printf("消息发送成功！响应结果：Code=%d, Msg=%s\n", resp.CodeError.Code, resp.CodeError.Msg)

	// 7. 按参数控制退出延时（新增核心功能）
	handleExitDelay(finalConfig.ExitDelay)
}

// maskString 脱敏字符串（仅保留前4位和后4位），避免密钥明文泄露
func maskString(s string) string {
	if len(s) <= 8 {
		return "******"
	}
	return s[:4] + "******" + s[len(s)-4:]
}

// handleExitDelay 处理退出延时逻辑
func handleExitDelay(delay int) {
	if delay <= 0 {
		fmt.Println("程序将直接退出...")
		return
	}
	fmt.Printf("程序将在 %d 秒后退出...\n", delay)
	time.Sleep(time.Duration(delay) * time.Second)
}
