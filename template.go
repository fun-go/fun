package fun

func genDefaultServiceTemplate() string {
	return `import client,{resultStatus,result,on} from "./client";
{{- range .GenServiceList}}
import {{.ServiceName}} from "./{{.ServiceName}}";
{{- end}}
export class defaultApi extends client {
  constructor(url: string) {
    super(url);
  }
  {{- range .GenServiceList}}
  public {{.ServiceName}}: {{.ServiceName}} = new {{.ServiceName}}(this);
  {{- end}}
}
export default class fun {
  static defaultApi: defaultApi | null = null
  static create(url: string): defaultApi {
    this.defaultApi = this.defaultApi ? this.defaultApi : new defaultApi (url);
    return this.defaultApi;
  }
}
export { resultStatus  };
export { result  };
export { on };`
}

func genServiceTemplate() string {
	return `import {defaultApi,result{{- if .IsIncludeProxy }},on{{- end}}} from "./fun"
{{- range .GenImport}}
import type {{.Name}} from "./{{.Path}}";
{{- end}}
export default class {{.ServiceName}} {
  private client: defaultApi;
  constructor(client: defaultApi) {
    this.client = client;
  }
  {{- $serviceName := .ServiceName }}
  {{- range .GenMethodTypeList}}
  async {{.MethodName}}({{.DtoText}}): Promise<{{.ReturnValueText}}> {
    return await this.client.request<{{.GenericTypeText}}>("{{$serviceName}}", "{{.MethodName}}"{{.ArgsText}})
  }
  {{- end}}
}`
}

func genStructTemplate() string {
	return `{{- range .GenImport}}import type {{.Name}} from "./{{.Path}}";{{"\n"}}{{- end}}export default interface {{.Name}} {
  {{- range .GenClassFieldType}}
  {{.Name}}:{{.Type}}
  {{- end}}
}`
}

func genEnumTemplate() string {
	return `enum {{.Name}} {
{{- range $index, $element := .Names}}
  {{$element}},
{{- end}}
}
{{if .DisplayNames}}export function {{.Name}}DisplayName(value:{{.Name}}): string {
  switch (value) {
{{- range $index, $element := .Names}}
    case {{$.Name}}.{{$element}}:
      return '{{index $.DisplayNames $index}}';
{{- end}}
    default:
      return "未知";
  }
}
{{end}}export default {{.Name}}`
}
