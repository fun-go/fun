export enum resultStatus {
    success,
    callError,
    error,
    closeError,
    networkError,
    timeoutError
}

export interface result<T> {
    id: string
    code: number | null
    msg: string | null
    status: resultStatus
    data: T
}

enum status {
    susses,
    close
}

enum workerDataType {
    open,
    close,
    data
}

interface workerType {
    type: workerDataType
    data: any
}

interface requestInfo<T> {
    id: string
    methodName: string
    serviceName: string
    dto?: T | undefined
    state?: { [key: string]: string }
    type: requestType
    func?: (data: result<any>) => void
    on?: on<any>
}

enum requestType {
    funcType,
    proxyType,
    closeType,
    initType
}

export class on<T> {
    onMessage(_: T): void {}

    onClose(): void {}
}

export default class client {
    private worker: globalThis.MessagePort | Worker | null = null;
    private status: status = status.close;
    private requestList: requestInfo<any>[] = [];
    private formerCall: ((serviceName: string ,methodName: string, state: Map<string, string>) => void) | null = null;
    private afterCall: (( serviceName: string ,methodName: string,result: result<any>) => result<any>) | null = null;
    private openCall: (() => void)[] = [];
    private closeCall: (() => void)[] = [];

    constructor(url: string) {
        this.worker = getWorker(url)
        const that = this;
        this.worker.onmessage = function (this: MessagePort, ev: MessageEvent) {
            const data: workerType = JSON.parse (ev.data);
            that.getMessage(data)
        }
        if (this.worker instanceof MessagePort) {
            window.addEventListener ('beforeunload', () => {
                this.requestList.forEach ((request) => {
                    if (request.type === requestType.proxyType) {
                        request.func && request.func (this.networkError(request.serviceName, request.methodName))
                        const map = new Map<string, any> ();
                        this.formerCall && this.formerCall (request.serviceName, request.methodName, map);
                        request.type = requestType.closeType;
                        request.state = Object.fromEntries(map);
                        this.worker?.postMessage (JSON.stringify (request));
                    }
                })
            });
        }
        this.worker instanceof MessagePort && this.worker.start ()
    }

    private getMessage(data: workerType) {
        const that = this;
        switch (data.type) {
            case workerDataType.open:
                that.status = status.susses
                that.requestList.forEach ((requestInfo) => {
                    that.worker?.postMessage (JSON.stringify (requestInfo));
                })
                that.openCall.forEach ((openCall) => {
                    openCall ()
                })
                break
            case workerDataType.close:
                that.status = status.close
                that.requestList.forEach ((request) => {
                    if (request.type === requestType.funcType) {
                        request.func && request.func (this.networkError(request.serviceName, request.methodName))
                    } else {
                        request.on?.onClose ()
                    }
                })
                that.requestList.length = 0
                that.closeCall.forEach ((closeCall) => {
                    closeCall ()
                })
                break;
            default:
                let result: result<any> = JSON.parse(data.data);
                this.getData(result)
        }
    }

    private getData(data: result<any>) {
        const index = this.requestList.findIndex ((request) => request.id === data.id)
        const request: requestInfo<any> = this.requestList[index]
        if (request.type === requestType.funcType) {
            request.func && request.func (this.after(request.serviceName, request.methodName,data))
        } else {
            if (data.status === resultStatus.success) {
                request.on?.onMessage (data.data)
            }  else  {
                request.on?.onClose ()
            }
        }
        (request.type === requestType.funcType || data.status !== resultStatus.success) && this.deleteRequest(request.id)
    }

    private networkError(serviceName: string ,methodName: string):result<any> {
        return this.after(serviceName, methodName,{id:"",status: resultStatus.networkError,data: null,msg: null,code: null})
    }

    private timeoutError(serviceName: string ,methodName: string):result<any> {
        return this.after(serviceName, methodName,{id:"",status: resultStatus.timeoutError,data: null,msg: null,code: null})
    }

    private after<T>(serviceName: string ,methodName: string, result: result<T>):result<T> {
        return this.afterCall ? this.afterCall (serviceName, methodName, result) : result
    }
    public async request<T>(serviceName: string, methodName: string, dto?: object | null): Promise<result<T>>;
    public async request<T>(serviceName: string, methodName: string, dto: object | null, on: on<T>): Promise<(() => void)>;
    public async request<T>(serviceName: string, methodName: string, dto: object | null = null, on?: on<T>): Promise<result<T> | (() => void)> {
        return new Promise<result<T> | (()=>void)>((resolve) => {
            const id: string = generateUUIDv4();
            const state = new Map<string, any> ();
            let requestInfo: requestInfo<any> = {
                id,
                methodName: methodName,
                serviceName: serviceName,
                type: on ? requestType.proxyType : requestType.funcType,
            };
            if (on) {
                requestInfo.on = on;
            } else {
                requestInfo.func = (data: result<T>) => {
                    resolve (this.after (serviceName, methodName, data));
                };
            }
            if (dto) {
                requestInfo.dto = dto;
            }
            this.formerCall && this.formerCall (serviceName, methodName, state);
            requestInfo.state = Object.fromEntries(state)
            if (this.status !== status.close) {
                this.worker?.postMessage (JSON.stringify (requestInfo));
            }
            this.requestList.push (requestInfo);
            const handleTimeout = (isNetworkError: boolean, timeout: number) => {
                setTimeout(() => {
                    const expectedStatus = isNetworkError ? status.close : status.susses;
                    if (this.status === expectedStatus && this.isRequestId(id)) {
                        resolve(isNetworkError ? this.networkError(serviceName, methodName) : this.timeoutError(serviceName, methodName));
                        this.deleteRequest(id);
                    }
                }, timeout);
            };
            if (!on) {
                handleTimeout(true, 2000);   // 网络错误
                handleTimeout(false, 10000); // 超时错误
            }
            if (on) {
                resolve(() => {
                    const state = new Map<string, any> ();
                    let requestInfo: requestInfo<any> = {
                        id,
                        methodName: methodName,
                        serviceName: serviceName,
                        type: requestType.closeType
                    };
                    on.onClose();
                    this.formerCall && this.formerCall (serviceName, methodName, state);
                    requestInfo.state = Object.fromEntries(state)
                    if (this.status !== status.close) {
                        this.worker?.postMessage (JSON.stringify (requestInfo));
                        this.requestList.push (requestInfo);
                    }
                })
            }
        })
    }

    private deleteRequest(id: string) {
        this.requestList = this.requestList.filter ((requestInfo) => {
            return requestInfo.id !== id
        })
    }

    private isRequestId(id: string) {
        return this.requestList.some ((requestInfo) => {
            return requestInfo.id === id
        })
    }


    public onFormer(func: (serviceName: string, methodName: string, state: Map<string, string>) => void) {
        this.formerCall = func
    }

    public onAfter(func: (serviceName: string ,methodName: string,result: result<any>) => result<any>) {
        this.afterCall = func
    }

    public onClose(func: () => void) {
        this.closeCall.push (func)
    }

    public onOpen(func: () => void) {
        this.openCall.push (func)
    }

}

const getWorker = (url: string):MessagePort | Worker => {
    const workerUrl = new URL( './worker', import.meta.url);
    let worker: MessagePort | Worker;
    if (typeof SharedWorker !== 'undefined') {
        worker = new SharedWorker(workerUrl).port;
    } else {
        worker =  new Worker(workerUrl);
    }
    worker.postMessage(worker.postMessage(JSON.stringify({ type: requestType.initType, id: getId(), data: url })));
    return worker;
};


function getId() : string {
    if (!localStorage.getItem ("id")) {
        localStorage.setItem ("id", generateUUIDv4())
    }
    return localStorage.getItem ("id") as string
}

function generateUUIDv4(): string {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0;
        const v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}