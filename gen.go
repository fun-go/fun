package fun

import (
	"bytes"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

type Gen interface {
	// TypeToTemplateType 将Go类型转换为模板中的类型表示
	typeToTemplateType(t reflect.Type) string

	// GenService 生成服务代码
	genService(service *service, serviceContext *genServiceType)

	// GenDefaultService 生成默认服务代码
	genDefaultService()

	// GenStruct 生成结构体代码
	genStruct(t reflect.Type) *genImportType

	// GetEnum 生成枚举代码
	getEnum(t reflect.Type) *genImportType

	// GetName 获取语言名称
	getName() string
}

type genType struct {
	GenServiceList []*genServiceType
}

type genMethodType struct {
	MethodName      string
	ReturnValueText string
	DtoText         string
	ArgsText        string
	GenericTypeText string
	IsProxy         bool
}

type genEnumType struct {
	Names        []string
	DisplayNames []string
	Name         string
}

type genImportType struct {
	Name string
}

type genServiceType struct {
	ServiceName       string
	GenMethodTypeList []*genMethodType
	GenImport         []*genImportType
	IsIncludeProxy    bool
}

type genClassType struct {
	Name              string
	GenImport         []*genImportType
	GenClassFieldType []*genClassFieldType
}

type genClassFieldType struct {
	Name string
	Type string
}

func deduplicateServiceImports(imports []*genImportType) []*genImportType {
	seen := make(map[string]bool)
	var result []*genImportType

	for _, imp := range imports {
		if !seen[imp.Name] {
			seen[imp.Name] = true
			result = append(result, imp)
		}
	}

	return result
}

func parseGenericTypeParams(typeName string) string {
	// 查找第一个 '[' 的位置
	start := strings.Index(typeName, "[")
	// 查找最后一个 ']' 的位置
	end := strings.LastIndex(typeName, "]")

	// 提取中括号内的内容
	paramsStr := typeName[start+1 : end]

	// 简单处理，按逗号分割（不考虑嵌套情况）
	params := strings.Split(paramsStr, ",")

	// 去除空格
	for i, param := range params {
		LL := strings.Split(strings.TrimSpace(param), ".")
		params[i] = firstLetterToUpper(LL[len(LL)-1])
	}

	return strings.Join(params, "")
}

func getGenericTypeName(typeName string) string {
	// 查找第一个 '[' 的位置
	start := strings.Index(typeName, "[")

	// 提取中括号内的内容
	paramsStr := typeName[0:start]

	return paramsStr
}

func genCode(templateContent string, outputFileName string, templateData any, languageName string) {
	tmpl, err := template.New(languageName).Parse(templateContent)
	if err != nil {
		panic(err.Error())
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		panic(err.Error())
	}
	code := buf.Bytes()
	fullPath := path.Join(directory, languageName)

	_, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(fullPath, os.ModePerm)
		if err != nil {
			panic(err.Error())
		}
	}
	err = os.WriteFile(path.Join(fullPath, outputFileName+"."+languageName), code, 0644)
	if err != nil {
		panic(err.Error())
	}
}

func firstLetterToLower(s string) string {
	if len(s) == 0 {
		return s
	}
	// 将字符串转换为rune切片以正确处理Unicode字符
	runes := []rune(s)
	// 将第一个rune转为小写
	runes[0] = unicode.ToLower(runes[0])

	// 转换回字符串并返回
	return string(runes)
}

func firstLetterToUpper(s string) string {
	if len(s) == 0 {
		return s
	}
	// 将字符串转换为rune切片以正确处理Unicode字符
	runes := []rune(s)
	// 将第一个rune转为大写
	runes[0] = unicode.ToUpper(runes[0])

	// 转换回字符串并返回
	return string(runes)
}

var directory = "./gen"

func SetOutput(path string) {
	directory = path
}

func camelToSnake(s string) string {
	// 在大写字母前添加下划线
	re := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	snake := re.ReplaceAllString(s, `${1}_${2}`)
	return strings.ToLower(snake)
}
