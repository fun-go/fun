package fun

type templateGo struct{}

func (ctx templateGo) genDefaultServiceTemplate() string {
	return `package api
import "github.com/fun-go/fun-client-go"
type Api struct {
{{- range .GenServiceList}}
	{{.ServiceName}} *{{.ServiceName}}
{{- end}}
    *client.Client
}
func CreateApi(url string) Api {
    apiClient := client.NewClient(url)
	return Api{
{{- range .GenServiceList}}
	    {{.ServiceName}}:New{{.ServiceName}}(apiClient),
{{- end}}
		Client:apiClient,
	}
}`
}

func (ctx templateGo) genServiceTemplate() string {
	return `package api
import "github.com/fun-go/fun-client-go"
type {{.ServiceName}} struct {
    *client.Client
}
func New{{.ServiceName}}(client *client.Client) *{{.ServiceName}} {
    return &{{.ServiceName}}{
        Client:client,
    }
}
{{- $serviceName := .ServiceName }}
{{- range .GenMethodTypeList}}
{{if eq .ArgsText ",dto,on"}}func (ctx *{{$serviceName}}) {{.MethodName}}({{.DtoText}}) {{.ReturnValueText}} {
    return client.Proxy[{{.GenericTypeText}}](ctx.Client,"{{$serviceName}}", "{{.MethodName}}"{{.ArgsText}})
}{{else}}func (ctx *{{$serviceName}}) {{.MethodName}}({{.DtoText}}) {{.ReturnValueText}} {
    return client.Request[{{.GenericTypeText}}](ctx.Client,"{{$serviceName}}", "{{.MethodName}}"{{.ArgsText}})
}{{end}}
{{- end}}`
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
