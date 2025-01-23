package core;

import (
	"fmt"
	"log"
	"errors"
	"reflect"
)

type ParamValue struct {
	VStr   string
	VInt64 int64
	VBool  bool
}

type Parameter struct {
	TypeStr      string
	Shortcut     string
	Name         string
	Value        ParamValue
	DefaultValue ParamValue
	Description  string
}

func (p *Parameter) IsDefault() bool {
	if p.TypeStr == "String" {
		return p.Value.VStr == p.DefaultValue.VStr
	}
	if p.TypeStr == "Int64" {
		return p.Value.VInt64 == p.DefaultValue.VInt64
	}
	if p.TypeStr == "Bool" {
		return p.Value.VBool == p.DefaultValue.VBool
	}
	return false
}

func (p *Parameter) GetJsonValue() string {
	if p.TypeStr == "String" {
		return "\"" + p.Value.VStr + "\""
	}
	if p.TypeStr == "Int64" {
		return fmt.Sprintf("%d", p.Value.VInt64)
	}
	if p.TypeStr == "Bool" {
		if p.Value.VBool {
			return "true"
		} else {
			return "false"
		}
	}
	return "null"
}

func MakeParameter(typeStr string, shortcut string, name string, value interface{}, description string) Parameter {
	p := Parameter{}
	givenType := ""
	switch value.(type) {
	case string:
		givenType = "String"
		p.Value.VStr = value.(string)
		p.DefaultValue.VStr = p.Value.VStr
	case int64:
		givenType = "Int64"
		p.Value.VInt64 = value.(int64)
		p.DefaultValue.VInt64 = p.Value.VInt64
	case int:
		givenType = "Int64"
		p.Value.VInt64 = int64(value.(int))
		p.DefaultValue.VInt64 = p.Value.VInt64
	case bool:
		givenType = "Bool"
		p.Value.VBool = value.(bool)
		p.DefaultValue.VBool = p.Value.VBool
	default:
		log.Fatal(errors.New("Unsupported parameter type " + reflect.TypeOf(value).Name()))
	}
	if typeStr != givenType {
		log.Fatal(errors.New("Type mismatch: given type is " + typeStr + ", given value is a " + givenType))
	}

	p.TypeStr = typeStr
	p.Shortcut = shortcut
	p.Name = name
	p.Description = description
	return p
}

func FindParameterValue(list []Parameter, name string) *ParamValue {
	for i := 0; i < len(list); i++ {
		if list[i].Name == name {
			return &list[i].Value
		}
	}
	return nil
}

func AddParameterToFlags(cmdFlags interface{}, inputs []reflect.Value, paramList []Parameter, idx int) {
	shortcutInc := 0
	var methodSuffix string
	if len(paramList[idx].Shortcut) > 0 {
		inputs[2] = reflect.ValueOf(paramList[idx].Shortcut)
		shortcutInc = 1
		methodSuffix = "VarP"
	} else {
		methodSuffix = "Var"
	}

	typeName := paramList[idx].TypeStr
	switch typeName {
	case "String":
		inputs[0] = reflect.ValueOf(&paramList[idx].Value.VStr)
		inputs[2+shortcutInc] = reflect.ValueOf(paramList[idx].Value.VStr)
	case "Int64":
		inputs[0] = reflect.ValueOf(&paramList[idx].Value.VInt64)
		inputs[2+shortcutInc] = reflect.ValueOf(paramList[idx].Value.VInt64)
	case "Bool":
		inputs[0] = reflect.ValueOf(&paramList[idx].Value.VBool)
		inputs[2+shortcutInc] = reflect.ValueOf(paramList[idx].Value.VBool)
	default:
		log.Fatal(errors.New("Unrecognised type " + typeName))
	}
	inputs[1] = reflect.ValueOf(paramList[idx].Name)
	inputs[3+shortcutInc] = reflect.ValueOf(paramList[idx].Description)

	nParams := len(inputs) - 1 + shortcutInc
	reflect.ValueOf(cmdFlags).MethodByName(typeName + methodSuffix).Call(inputs[0:nParams])
}
