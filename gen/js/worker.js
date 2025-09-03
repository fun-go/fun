let url = "";
let id = "";
let ws = null;
let clientList = [];
let requestList = [];
let time1 = null;
// 判断是否是 SharedWorker
if (typeof SharedWorkerGlobalScope !== 'undefined' && self instanceof SharedWorkerGlobalScope) {
    // SharedWorker 版本
    self.onconnect = function (e) {
        const port = e.ports[0];
        if (ws &&time1){
            port.postMessage(JSON.stringify({ type: 0 }));
        }
        clientList.push(port);

        // 监听消息
        port.onmessage = handleMessage(port)

        port.start();
    };
} else {
    // 普通 Worker 版本
    self.onmessage = handleMessage()
    newWs();
}

function ping(ws) {
    const binaryData = new Uint8Array([0]);
    ws.send(binaryData.buffer);
}

function handleMessage(port) {
    return function (e) {
        const data = JSON.parse(e.data);
        if (data.type === 3) {
            if (!ws) {
                url = data.data;
                id = data.id;
                newWs();
            }
            return
        }else if (data.type === 2) {
            requestList = requestList.filter ((requestInfo) => {
                return requestInfo.request.id !== data.id;
            })
        } else {
            requestList.push({
                request: data,
                port:port
            });
        }
        ws.send(JSON.stringify(data));
    };
}
function newWs() {
    ws = new WebSocket(url + "?id=" + id);
    let time = null;

    ws.onopen = function () {
        if (clientList.length !== 0) {
            clientList.forEach((port) => {
                port.postMessage(JSON.stringify({
                    type: 0
                }))
            });
        } else {
            self.postMessage(JSON.stringify({ type: 0 }));
        }

        const Ping = () => {
            ping(ws);
            time = setTimeout(() => {
                ws.close();
            }, 2000);
        };

        Ping();
        time1 = setInterval(() => {
            Ping();
        }, 5000);
    };

    ws.onmessage = function (evt) {
        if (typeof evt.data === 'string') {
            const data = JSON.parse(evt.data);
            const index = requestList.findIndex((request) => request.request.id === data.id);
            const request = requestList[index];
            if (request) {
                if (request.type === 0 || data.status === 3) {
                    requestList.splice(index, 1);
                }
                if (request.port) {
                    request.port.postMessage(JSON.stringify({ type: 2, data: JSON.stringify(data) }));
                } else {
                    self.postMessage(JSON.stringify({ type: 2, data: JSON.stringify(data) }));
                }
            }
        } else if (evt.data instanceof Blob) {
            var blobReader = new Response(evt.data).bytes()
            blobReader.then(res => {
                if (res[0] === 1) {
                    clearTimeout(time);
                }
            })
        }
    };

    ws.onclose = function () {
        console.log("WebSocket 已关闭")
        if (time1) {
            if (clientList.length !== 0) {
                clientList.forEach((p) => {
                    p.postMessage(JSON.stringify({ type: 1 }));
                });
            } else {
                self.postMessage(JSON.stringify({ type: 1 }));
            }
            requestList.length = 0;
            clearInterval(time1);
            newWs(); // 重新连接
        } else {
            setTimeout(() => {
                newWs(); // 重新连接
            }, 5000);
        }
    };
}