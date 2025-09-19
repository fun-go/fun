package fun

import (
	"reflect"

	"github.com/gorilla/websocket"
)

// 普通发送信息
func (fun *Fun) send(id string, text any) bool {
	return fun.getConnInfoAndSend(id, func(loadConnInfo connInfoType) error {
		return loadConnInfo.conn.WriteJSON(text)
	})
}

// 推送
func (fun *Fun) push(id string, requestId string, data any) bool {
	connInfo, ok := fun.connList.Load(id)
	if !ok {
		return false
	}
	loadConnInfo := connInfo.(connInfoType)
	on, ok := loadConnInfo.onList.Load(requestId)
	if !ok {
		return false
	}

	method := fun.serviceList[on.(onType).serviceName].methodList[on.(onType).methodName]
	if method.method.Type.Out(0).Elem() != reflect.TypeOf(data) {
		return false
	}
	// 准备数据
	result := success(data)
	result.Id = requestId
	map1 := toLowerMap(result)

	// 只在实际发送时加锁
	loadConnInfo.mu.Lock()
	err := loadConnInfo.conn.WriteJSON(map1)
	loadConnInfo.mu.Unlock()
	return err == nil
}

// 发送二进制信息
func (fun *Fun) sendBinary(id string, data []byte) bool {
	return fun.getConnInfoAndSend(id, func(loadConnInfo connInfoType) error {
		return loadConnInfo.conn.WriteMessage(websocket.BinaryMessage, data)
	})
}

func (fun *Fun) sendPong(id string) {
	fun.sendBinary(id, []byte{1})
}

// 发送前统一加锁处理 避免同时发送冲突
func (fun *Fun) getConnInfoAndSend(id string, callback func(loadConnInfo connInfoType) error) bool {
	if connInfo, ok := fun.connList.Load(id); ok {
		loadConnInfo := connInfo.(connInfoType)
		loadConnInfo.mu.Lock()
		err := callback(loadConnInfo)
		loadConnInfo.mu.Unlock()
		return err == nil
	}
	return false
}
