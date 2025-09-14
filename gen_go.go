package fun

import (
	"reflect"
	"strings"
)

type GenGo struct {
	template templateGo
}

func (ctx GenGo) typeToTemplateType(t reflect.Type) string {
	text := ""
	if t.Kind() == reflect.Ptr {
		text += "*"
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Slice:
		text += "[]" + ctx.typeToTemplateType(t.Elem())
		break
	default:
		text += t.Name()
	}
	return text
}

func (ctx GenGo) genService(
	service *service,
	serviceContext *genServiceType,
) {
	// 遍历服务中的每个方法
	isIncludeProxy := false
	for _, method := range service.methodList {
		var returnValueText string
		var dtoText string
		var argsText string
		var genericTypeText string

		if method.isProxy {
			isIncludeProxy = true
		}
		if method.method.Type.NumOut() == 0 {
			returnValueText = "null"
		} else {
			returnType := method.method.Type.Out(0)

			// 转换为 TypeScript 类型
			// 创建结构体模板数据
			//处理泛类
			t := ctx.typeToTemplateType(returnType)
			if !strings.Contains(t, "[]") && strings.Contains(t, "[") {
				returnValueText = getGenericTypeName(t) + parseGenericTypeParams(t)
			} else {
				returnValueText = t
			}

			// 如果是结构体类型，递归生成导入路径
			if returnType.Kind() == reflect.Ptr {
				returnType = returnType.Elem()
			}
			if returnType.Kind() == reflect.Struct {
				ctx.genStruct(returnType)
			}
			if returnType.Kind() == reflect.Slice {
				fieldType := returnType.Elem()
				if fieldType.Kind() == reflect.Ptr {
					fieldType = fieldType.Elem()
				}
				if fieldType.Kind() == reflect.Struct {
					ctx.genStruct(fieldType)
				}
			}
			enumType := reflect.TypeOf((*enum)(nil)).Elem()
			displayEnumType := reflect.TypeOf((*displayEnum)(nil)).Elem()
			if returnType.Kind() == reflect.Uint8 && (returnType.Implements(displayEnumType) || returnType.Implements(enumType)) {
				ctx.getEnum(returnType)
			}
		}

		// 处理 DTO 参数
		if method.dto != nil {
			v := ctx.typeToTemplateType(*method.dto)
			if !strings.Contains(v, "[]") && strings.Contains(v, "[") {
				dtoText += "dto:" + getGenericTypeName(v) + parseGenericTypeParams(v)
			} else {
				dtoText += "dto:" + v
			}
			argsText += ",dto"
			ctx.genStruct(*method.dto)
		}

		// 处理代理逻辑（on 回调）
		if method.isProxy {
			if method.dto != nil {
				dtoText += ","
			}
			dtoText += "on:on<" + strings.ReplaceAll(returnValueText, " | null", "") + ">"
			argsText += ",on"
			genericTypeText = strings.ReplaceAll(returnValueText, " | null", "")
			returnValueText = "() => void"
		} else {
			genericTypeText = returnValueText
			returnValueText = "result<" + returnValueText + ">"
		}

		// 添加方法信息到服务上下文
		serviceContext.GenMethodTypeList = append(serviceContext.GenMethodTypeList, &genMethodType{
			MethodName:      method.method.Name,
			ReturnValueText: returnValueText,
			DtoText:         dtoText,
			ArgsText:        argsText,
			GenericTypeText: genericTypeText,
		})
	}
	serviceContext.IsIncludeProxy = isIncludeProxy
	// 去重导入路径
	// 生成 TypeScript 文件
	genCode(
		ctx.template.genServiceTemplate(),
		camelToSnake(service.serviceType.Name()),
		serviceContext,
		ctx.getName(),
	)
}

func (ctx GenGo) genDefaultService() {
	f := GetFun()
	genContext := genType{GenServiceList: []*genServiceType{}}

	for _, service := range f.serviceList {
		serviceContext := &genServiceType{
			ServiceName:       service.serviceType.Name(),
			GenMethodTypeList: []*genMethodType{},
		}

		genContext.GenServiceList = append(genContext.GenServiceList, serviceContext)
		ctx.genService(service, serviceContext)
	}
	genCode(ctx.template.genDefaultServiceTemplate(), "fun", genContext, ctx.getName())
}

func (ctx GenGo) genStruct(t reflect.Type) *genImportType {

	var structTemplate genClassType
	// 创建结构体模板数据
	if !strings.Contains(t.String(), "[]") && strings.Contains(t.String(), "[") {
		structTemplate = genClassType{
			Name: getGenericTypeName(t.Name()) + parseGenericTypeParams(t.Name()),
		}
	} else {
		structTemplate = genClassType{
			Name: t.Name(),
		}
	}

	// 遍历结构体字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type

		jsType := ctx.typeToTemplateType(fieldType)
		name := field.Name
		// 生成字段类型并添加到模板
		if !strings.Contains(jsType, "[]") && strings.Contains(jsType, "[") {
			structTemplate.GenClassFieldType = append(structTemplate.GenClassFieldType, &genClassFieldType{
				Name: name,
				Type: getGenericTypeName(jsType) + parseGenericTypeParams(jsType),
			})
		} else {
			structTemplate.GenClassFieldType = append(structTemplate.GenClassFieldType, &genClassFieldType{
				Name: name,
				Type: jsType,
			})
		}

		// 如果字段是结构体，递归生成导入路径
		if fieldType.Kind() == reflect.Struct {
			ctx.genStruct(fieldType)
		}

		if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Struct {
			ctx.genStruct(fieldType.Elem())
		}

		enumType := reflect.TypeOf((*enum)(nil)).Elem()
		displayEnumType := reflect.TypeOf((*displayEnum)(nil)).Elem()

		if fieldType.Kind() == reflect.Uint8 && (fieldType.Implements(displayEnumType) || fieldType.Implements(enumType)) {
			ctx.getEnum(fieldType)
		}

	}

	// 生成 TypeScript 文件
	genCode(
		ctx.template.genStructTemplate(),
		camelToSnake(structTemplate.Name),
		structTemplate,
		ctx.getName(),
	)
	return &genImportType{}
}

func (ctx GenGo) getEnum(t reflect.Type) *genImportType {
	var enumTemplate genEnumType
	displayEnumType := reflect.TypeOf((*displayEnum)(nil)).Elem()
	statusValue := reflect.New(t).Elem()
	if t.Implements(displayEnumType) {
		enumValue := statusValue.Interface().(displayEnum)
		enumTemplate.Names = enumValue.Names()
		enumTemplate.DisplayNames = enumValue.DisplayNames()
	} else {
		enumValue := statusValue.Interface().(enum)
		enumTemplate.Names = enumValue.Names()
	}
	enumTemplate.Name = t.Name()

	genCode(
		ctx.template.genEnumTemplate(),
		camelToSnake(t.Name()),
		enumTemplate,
		ctx.getName(),
	)

	return &genImportType{}
}

func (ctx GenGo) getName() string {
	return "go"
}
