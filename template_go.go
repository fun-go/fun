package fun

type templateGo struct{}

func (ctx templateGo) genDefaultServiceTemplate() string {
	return `package api
type Api struct {
{{- range .GenServiceList}}
	{{.ServiceName}} *{{.ServiceName}}
{{- end}}
    *Client
}
func CreateApi(url string) Api {
    client := NewClient(url)
	return Api{
{{- range .GenServiceList}}
	    {{.ServiceName}}:New{{.ServiceName}}(client),
{{- end}}
		Client:client,
	}
}`
}

func (ctx templateGo) genServiceTemplate() string {
	return `package api
type {{.ServiceName}} struct {
    *Client
}
func New{{.ServiceName}}(client *Client) *{{.ServiceName}} {
    return &{{.ServiceName}}{
        Client:client,
    }
}
{{- $serviceName := .ServiceName }}
{{- range .GenMethodTypeList}}
func (ctx *{{$serviceName}}) {{.MethodName}}({{.DtoText}}) {{.ReturnValueText}} {
    return await this.client.request<{{.GenericTypeText}}>("{{$serviceName}}", "{{.MethodName}}"{{.ArgsText}})
}{{- end}}`
}

func (ctx templateGo) genStructTemplate() string {
	return `package api
type {{.Name}} struct{
  {{- range .GenClassFieldType}}
    {{.Name}} {{.Type}}
  {{- end}}
}`
}

func (ctx templateGo) genEnumTemplate() string {
	return `package api
type {{.Name}} uint8{{$enumName := .Name}}
const (
{{- range $index, $element := .Names}} 
    {{$element}}{{if eq $index 0}}        {{$enumName}} = iota{{end}}
{{- end}}
)
func ({{.Name}}) Values() []{{.Name}} {
	return []{{.Name}}{
{{- range $index, $element := .Names}}
        {{$element}},
{{- end}}
	}
}{{if .DisplayNames}}
func ({{.Name}}) DisplayNames() []string {
	return []string{
{{- range $index, $element := .DisplayNames}}
        "{{$element}}",
{{- end}}
	}
}{{end}}`
}
