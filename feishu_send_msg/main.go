package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/core"
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
}

func main() {
	// 1. 定义命令行参数（所有参数都是可选的，传了就用，没传就读配置文件）
	var (
		appID         = flag.String("app-id", "", "飞书应用ID")
		appSecret     = flag.String("app-secret", "", "飞书应用密钥")
		receiveIdType = flag.String("receive-id-type", "", "接收ID类型")
		receiveId     = flag.String("receive-id", "", "接收ID")
		content       = flag.String("content", "", "发送的文本内容")
		uuid          = flag.String("uuid", "", "消息UUID")
	)
	flag.Parse()

	// 2. 初始化最终配置
	var finalConfig Config

	// 3. 判断是否传入了命令行核心参数（只要传了任意核心参数，就优先用命令行）
	hasCLIParams := *appID != "" || *appSecret != "" || *receiveId != "" || *content != ""
	if hasCLIParams {
		// 命令行模式：校验必填参数必须全传
		if *appID == "" || *appSecret == "" || *receiveId == "" || *content == "" {
			fmt.Println("错误：命令行模式下必须传入所有核心参数！")
			fmt.Println("必填参数：-app-id、-app-secret、-receive-id、-content")
			fmt.Println("可选参数：-receive-id-type（默认email）、-uuid")
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
		}
		// 命令行模式下，receiveIdType默认值
		if finalConfig.ReceiveIdType == "" {
			finalConfig.ReceiveIdType = "email"
		}
		fmt.Println("使用命令行参数模式...")
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

		// 配置文件模式下，receiveIdType默认值
		if finalConfig.ReceiveIdType == "" {
			finalConfig.ReceiveIdType = "email"
		}
		fmt.Println("使用配置文件模式（config.json）...")
	}

	// 4. 创建飞书客户端并发送消息
	client := lark.NewClient(finalConfig.AppID, finalConfig.AppSecret)
	contentJSON := fmt.Sprintf(`{"text":"%s"}`, finalConfig.Content)

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
		os.Exit(1)
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Printf("请求失败 - logId: %s, 错误码: %d, 错误信息: %s\n",
			resp.RequestId(), resp.CodeError.Code, resp.CodeError.Msg)
		os.Exit(1)
	}

	// 成功提示
	fmt.Printf("消息发送成功！响应结果：%s\n", larkcore.Prettify(resp))
}
