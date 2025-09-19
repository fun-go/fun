package fun

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"
)

var testPort *uint16 = nil

var testRequestMap = sync.Map{} // map[string]*testRequestInfo

type testRequestInfo struct {
	resultChan chan Result[any]
	on         On[any]
}

func getMessage[T any](t *testing.T, id string) Result[T] {
	// 创建请求信息并存储到 map 中
	resultChan := make(chan Result[any], 1)
	requestInfo := testRequestInfo{
		resultChan: resultChan,
	}
	testRequestMap.Store(id, requestInfo)

	// 设置超时
	timeout := time.After(10 * time.Second)

	select {
	case result := <-resultChan:
		convertedData := safeConvert[*T](result.Data)
		// 清理请求信息
		return Result[T]{
			Id:     result.Id,
			Data:   convertedData,
			Status: result.Status,
			Code:   result.Code,
			Msg:    result.Msg,
		}
	case <-timeout:
		// 清理请求信息
		testRequestMap.Delete(id)
		ErrorLogger(callError("fun:request timeout"))
		t.Fail()
		return Result[T]{}
	}
}

func mockSendJson(t *testing.T, requestInfo any) {
	map1 := toLowerMap(requestInfo, t)
	writeMutex.Lock()
	_ = testClient.WriteJSON(map1)
	writeMutex.Unlock()
}

func MockRequest[T any](t *testing.T, requestInfo any) Result[T] {
	newClientOrService()
	requestId := reflect.ValueOf(requestInfo).FieldByName("Id").String()
	mockSendJson(t, requestInfo)
	return getMessage[T](t, requestId)
}

type On[T any] struct {
	Message func(message T)
	Close   func()
}

func MockProxyClose(t *testing.T, id string) {
	requestInfo := RequestInfo[any]{
		Id:   id,
		Type: CloseType,
	}
	mockSendJson(t, requestInfo)
}

func MockProxy[T any](t *testing.T, requestInfo any, on On[T], seconds int64) {
	newClientOrService()
	requestId := reflect.ValueOf(requestInfo).FieldByName("Id").String()

	// 创建代理请求信息并存储到 map 中
	onAny := On[any]{
		Message: func(message any) {
			// 类型断言转换为 T 类型
			convertedData := safeConvert[T](message)
			on.Message(convertedData)
		},
		Close: on.Close,
	}
	requestInfo = testRequestInfo{
		on: onAny,
	}
	testRequestMap.Store(requestId, requestInfo)

	mockSendJson(t, requestInfo)

	// 启动超时处理
	time.Sleep(time.Duration(seconds) * time.Second)
	if value, exists := testRequestMap.Load(requestId); exists {
		reqInfo := value.(testRequestInfo)
		if reqInfo.on.Close != nil {
			reqInfo.on.Close()
		}
		testRequestMap.Delete(requestId)
		// 发送关闭请求
		mockSendJson(t, RequestInfo[any]{
			Id:   requestId,
			Type: CloseType,
		})
	}
}

var clientOnce sync.Once

func newClientOrService() {
	// 使用 sync.Once 确保初始化代码只执行一次，且线程安全
	clientOnce.Do(func() {
		port := randomPort()
		testPort = &port
		go func() {
			testStart(port)
		}()
		mockClient(*testPort)
	})
}

func safeConvert[T any](data any) (result T) {
	defer func() {
		if r := recover(); r != nil {
			result = *new(T)
		}
	}()

	// 直接类型断言
	if typedData, typeOk := data.(T); typeOk {
		return typedData
	}

	if data == nil {
		return *new(T)
	}

	// 尝试通过JSON进行转换
	jsonData, err := json.Marshal(data)
	if err != nil {
		return *new(T)
	}

	var converted T
	err = json.Unmarshal(jsonData, &converted)
	if err != nil {
		return *new(T)
	}

	return converted
}
