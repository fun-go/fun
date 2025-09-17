package fun

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var clientId = "lGbk6IVcT965Qs_zb30KS"
var writeMutex sync.Mutex // 添加互斥锁

func SetTestClientId(id string) {
	clientId = id
}

var testClient *websocket.Conn = nil

func mockClient(port uint16) {
	url := fmt.Sprintf("ws://127.0.0.1:%d?id=%s", port, clientId)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	for err != nil {
		conn, _, err = websocket.DefaultDialer.Dial(url, nil)
		time.Sleep(100 * time.Millisecond)
	}
	testClient = conn
	go func() {
		for {
			writeMutex.Lock() // 加锁
			err = conn.WriteMessage(websocket.BinaryMessage, []byte{0})
			writeMutex.Unlock()
			time.Sleep(5 * time.Second)
		}
	}()
	go func() {
		for {
			messageType, message, _ := conn.ReadMessage()
			if messageType == websocket.TextMessage {
				var result Result[any]
				if err := json.Unmarshal(message, &result); err == nil {
					handleTestMessage(result)
				}
			}
		}
	}()
}

// 处理测试消息并分发给对应的请求
func handleTestMessage(result Result[any]) {
	// 检查是否有对应的测试请求
	if value, exists := testRequestMap.Load(result.Id); exists {
		requestInfo := value.(testRequestInfo)

		// 处理普通请求
		if requestInfo.resultChan != nil {
			requestInfo.resultChan <- result
		} else {
			if result.Status == closeErrorCode {
				if requestInfo.on.Close != nil {
					requestInfo.on.Close()
				}
				testRequestMap.Delete(result.Id)
			} else {
				if requestInfo.on.Message != nil {
					requestInfo.on.Message(result.Data)
				}
			}
		}
	}
}
