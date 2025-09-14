package fun

type templateTs struct{}

func (ctx templateTs) genDefaultServiceTemplate() string {
	return `import client from "fun-client";
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
export default class api {
  static create(url: string): defaultApi {
    return new defaultApi (url);
  }
}`
}

func (ctx templateTs) genServiceTemplate() string {
	return `import {defaultApi,type result{{- if .IsIncludeProxy }},on{{- end}}} from "fun-client"
{{- range .GenImport}}
import type {{.Name}} from "./{{.Name}}";
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

func (ctx templateTs) genStructTemplate() string {
	return `{{- range .GenImport}}import type {{.Name}} from "./{{.Name}}";{{"\n"}}{{- end}}export default interface {{.Name}} {
  {{- range .GenClassFieldType}}
  {{.Name}}:{{.Type}}
  {{- end}}
}`
}

func (ctx templateTs) genEnumTemplate() string {
	return `enum {{.Name}} {
{{- range $index, $element := .Names}}
  {{$element}},
{{- end}}
}{{$enumName := .Name}}
function values() :{{.Name}}[] {
	return [
{{- range $index, $element := .Names}}
        {{$enumName}}.{{$element}},
{{- end}}
	]
}{{if .DisplayNames}}
function displayNames() :string[] {
	return [
{{- range $index, $element := .DisplayNames}}
        "{{$element}}",
{{- end}}
	]
}{{end}}
export default {{.Name}}`
}
