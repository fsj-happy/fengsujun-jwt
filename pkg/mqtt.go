package pkg

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// MQTT配置信息
const (
	Broker = "14.103.243.150"
	Port   = 1883
	Topic  = "go/mqtt/test" // 这里填写你要订阅/发布的主题
)

// 连接成功回调函数
var ConnectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("连接成功")
}

// 连接丢失回调函数
var ConnectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("连接丢失: %v\n", err)
}

// 消息处理回调函数
var MessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("收到消息: %s 来自主题: %s\n", msg.Payload(), msg.Topic())
}

// CreateClient 创建MQTT客户端
func CreateClient(clientID string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", Broker, Port))
	opts.SetClientID(clientID)
	opts.SetConnectRetry(true)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(ConnectHandler)
	opts.SetConnectionLostHandler(ConnectLostHandler)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("连接失败: %v", token.Error())
	}
	return client
}

// StartPublisher 封装发布端逻辑
func StartPublisher(clientID string) {
	client := CreateClient(clientID)
	defer client.Disconnect(250)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Println("开始发布保留消息 (每5秒一次)，按Ctrl+C退出")

	for {
		select {
		case <-ticker.C:
			message := fmt.Sprintf("Hello MQTT! Time: %s", time.Now().Format("15:04:05"))
			token := client.Publish(Topic, 1, true, message) // 这里设置为true
			token.Wait()
			if token.Error() != nil {
				fmt.Printf("发布失败: %v\n", token.Error())
			} else {
				fmt.Printf("发布成功: %s\n", message)
			}
		case <-c:
			fmt.Println("\n接收到退出信号，正在断开连接...")
			return
		}
	}
}

// StartSubscriber 封装订阅端逻辑
func StartSubscriber(clientID string) {
	client := CreateClient(clientID)
	defer client.Disconnect(250)

	token := client.Subscribe(Topic, 1, MessageHandler)
	if token.Wait() && token.Error() != nil {
		log.Fatalf("订阅失败: %v", token.Error())
	}
	fmt.Printf("已订阅主题: %s\n", Topic)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\n接收到退出信号，正在断开连接...")
}
