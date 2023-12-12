package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	target = os.Getenv("TARGET_URL")
)

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 过滤无效URL
	_, err := http.NewRequest(request.HTTPMethod, request.Path, nil)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	// 去掉环境前缀（针对腾讯云，如果包含的话，目前我只用到了test和release）
	newPath := strings.Replace(request.Path, "/openai-go", "", 1)
	newPath = strings.Replace(newPath, "/default", "", 1)

	// 拼接目标URL
	targetURL := target + newPath

	// 创建代理HTTP请求
	proxyReq, err := http.NewRequest(request.HTTPMethod, targetURL, strings.NewReader(request.Body))
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	// 将原始请求头复制到新请求中
	for headerKey, headerValues := range request.Headers {
		for _, headerValue := range headerValues {
			proxyReq.Header.Add(headerKey, string(headerValue))
		}
	}

	// 默认超时时间设置为60s
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// 向 OpenAI 发起代理请求
	resp, err := client.Do(proxyReq)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}
	defer resp.Body.Close()

	// 将响应头复制到代理响应头中
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		var headerValue string
		for _, value := range values {
			headerValue += value + ","
		}
		responseHeaders[key] = strings.TrimSuffix(headerValue, ",")
	}

	// 读取响应实体到字节数组中
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	// 返回API Gateway响应
	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(responseBody),
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
