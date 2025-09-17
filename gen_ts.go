package fun

import (
	"reflect"
	"strings"
)

type GenTs struct {
	template templateTs
}

func (ctx GenTs) typeToTemplateType(t reflect.Type) string {
	text := ""
	if t.Kind() == reflect.Ptr {
		text += " | null"
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if t.Kind() == reflect.Uint8 {
			enumType := reflect.TypeOf((*enum)(nil)).Elem()
			displayEnumType := reflect.TypeOf((*displayEnum)(nil)).Elem()
			if t.Implements(displayEnumType) || t.Implements(enumType) {
				text = t.Name() + text
			} else {
				text = "number" + text
			}
		} else {
			text = "number" + text
		}
		break
	case reflect.Bool:
		text = "boolean" + text
		break
	case
		reflect.String, reflect.Struct:
		text = t.Name() + text
		break
	default:
		text = ctx.typeToTemplateType(t.Elem()) + "[]" + text
		break
	}
	return text
}

func (ctx GenTs) genService(
	service *service,
	serviceContext *genServiceType,
) {
	// 收集服务方法中涉及的结构体导入路径
	var nestedImports []*genImportType

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
			t := firstLetterToLower(ctx.typeToTemplateType(returnType))
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
				nestedImports = append(nestedImports, ctx.genStruct(returnType))
			}
			if returnType.Kind() == reflect.Slice {
				fieldType := returnType.Elem()
				if fieldType.Kind() == reflect.Ptr {
					fieldType = fieldType.Elem()
				}
				if fieldType.Kind() == reflect.Struct {
					nestedImports = append(nestedImports, ctx.genStruct(fieldType))
				}
			}
			enumType := reflect.TypeOf((*enum)(nil)).Elem()
			displayEnumType := reflect.TypeOf((*displayEnum)(nil)).Elem()
			if returnType.Kind() == reflect.Uint8 && (returnType.Implements(displayEnumType) || returnType.Implements(enumType)) {
				nestedImports = append(nestedImports, ctx.getEnum(returnType))
			}
		}

		// 处理 DTO 参数
		if method.dto != nil {
			v := firstLetterToLower(ctx.typeToTemplateType(*method.dto))
			if !strings.Contains(v, "[]") && strings.Contains(v, "[") {
				dtoText += "dto:" + getGenericTypeName(v) + parseGenericTypeParams(v)
			} else {
				dtoText += "dto:" + v
			}
			argsText += ",dto"
			nestedImports = append(nestedImports, ctx.genStruct(*method.dto))
		} else {
			if method.isProxy {
				argsText += ",null"
			}
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
			MethodName:      firstLetterToLower(method.method.Name),
			ReturnValueText: returnValueText,
			DtoText:         dtoText,
			ArgsText:        argsText,
			GenericTypeText: firstLetterToLower(genericTypeText),
			IsProxy:         method.isProxy,
		})
	}
	serviceContext.IsIncludeProxy = isIncludeProxy
	// 去重导入路径
	serviceContext.GenImport = deduplicateServiceImports(nestedImports)

	// 生成 TypeScript 文件
	genCode(
		ctx.template.genServiceTemplate(),
		firstLetterToLower(service.serviceType.Name()),
		serviceContext,
		ctx.getName(),
	)
}

func (ctx GenTs) genDefaultService() {
	f := GetFun()
	genContext := genType{GenServiceList: []*genServiceType{}}

	for _, service := range f.serviceList {
		serviceContext := &genServiceType{
			ServiceName:       firstLetterToLower(service.serviceType.Name()),
			GenMethodTypeList: []*genMethodType{},
		}

		genContext.GenServiceList = append(genContext.GenServiceList, serviceContext)
		ctx.genService(service, serviceContext)
	}
	genCode(ctx.template.genDefaultServiceTemplate(), "fun", genContext, ctx.getName())
}

func (ctx GenTs) genStruct(t reflect.Type) *genImportType {
	var structTemplate genClassType
	// 创建结构体模板数据
	if !strings.Contains(t.String(), "[]") && strings.Contains(t.String(), "[") {
		structTemplate = genClassType{
			Name: firstLetterToLower(getGenericTypeName(t.Name())) + parseGenericTypeParams(t.Name()),
		}
	} else {
		structTemplate = genClassType{
			Name: firstLetterToLower(t.Name()),
		}
	}
	// 收集嵌套结构体的导入路径
	var nestedImports []*genImportType

	// 遍历结构体字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type

		jsType := ctx.typeToTemplateType(fieldType)
		name := field.Name
		// 解引用指针
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
			name += "?"
		}
		// 生成字段类型并添加到模板
		if !strings.Contains(jsType, "[]") && strings.Contains(jsType, "[") {
			structTemplate.GenClassFieldType = append(structTemplate.GenClassFieldType, &genClassFieldType{
				Name: firstLetterToLower(name),
				Type: firstLetterToLower(getGenericTypeName(jsType)) + parseGenericTypeParams(jsType),
			})
		} else {
			structTemplate.GenClassFieldType = append(structTemplate.GenClassFieldType, &genClassFieldType{
				Name: firstLetterToLower(name),
				Type: firstLetterToLower(jsType),
			})
		}

		// 如果字段是结构体，递归生成导入路径
		if fieldType.Kind() == reflect.Struct {
			nestedImports = append(nestedImports, ctx.genStruct(fieldType))
		}

		if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Struct {
			nestedImports = append(nestedImports, ctx.genStruct(fieldType.Elem()))
		}

		enumType := reflect.TypeOf((*enum)(nil)).Elem()
		displayEnumType := reflect.TypeOf((*displayEnum)(nil)).Elem()

		if fieldType.Kind() == reflect.Uint8 && (fieldType.Implements(displayEnumType) || fieldType.Implements(enumType)) {
			nestedImports = append(nestedImports, ctx.getEnum(fieldType))
		}

	}

	// 去重并计算相对路径
	uniqueImports := deduplicateServiceImports(nestedImports)

	// 将去重后的导入路径添加到结构体模板中
	structTemplate.GenImport = uniqueImports

	// 生成 TypeScript 文件
	genCode(
		ctx.template.genStructTemplate(),
		structTemplate.Name,
		structTemplate,
		ctx.getName(),
	)

	if !strings.Contains(t.String(), "[]") && strings.Contains(t.String(), "[") {
		return &genImportType{
			Name: structTemplate.Name,
		}
	} else {
		return &genImportType{
			Name: firstLetterToLower(t.Name()),
		}
	}
}

func (ctx GenTs) getEnum(t reflect.Type) *genImportType {
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
	enumTemplate.Name = firstLetterToLower(t.Name())

	genCode(
		ctx.template.genEnumTemplate(),
		firstLetterToLower(t.Name()),
		enumTemplate,
		ctx.getName(),
	)

	return &genImportType{Name: firstLetterToLower(t.Name())}
}

func (ctx GenTs) getName() string {
	return "ts"
}
