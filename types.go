package main

import (
	"encoding/json"

	"github.com/iancoleman/strcase"
)

func Unmarshal(data []byte) ([]DataType, error) {
	type NodeType struct {
		Type     string               `json:"type"`
		Named    bool                 `json:"named"`
		Subtypes []Type               `json:"subtypes,omitempty"`
		Fields   map[string]JSONField `json:"fields,omitempty"`
		Children *JSONField           `json:"children,omitempty"`
	}

	type NodeTypes []NodeType

	var n NodeTypes
	err := json.Unmarshal(data, &n)
	if err != nil {
		return nil, err
	}

	var ret []DataType

	for _, k := range n {
		t := k.Type
		n := k.Named

		if len(k.Subtypes) == 0 {
			if len(k.Fields) == 0 && k.Children == nil {
				ret = append(ret, LeafType{
					CommonInfo{t, n},
				})
			} else {
				m := map[string]Field{}
				for k, v := range k.Fields {
					m[k] = *v.Into()
				}
				ret = append(ret, ProductType{
					CommonInfo{t, n},
					k.Children.Into(),
					m,
				})
			}
		} else {
			ret = append(ret, SumType{
				CommonInfo{t, n},
				k.Subtypes,
			})
		}
	}

	return ret, nil
}

type DataType interface {
	Name() string
	Named() bool
}

type DataTypes []DataType

func (d DataTypes) FindType(s string) DataType {
	for _, v := range d {
		if v.Name() == s {
			return v
		}
	}
	return nil
}

type OneOfPair struct {
	Name string
	Kind OneOfType
}

func (d DataTypes) OneOfTypesIncluding(s string) []OneOfPair {
	var ret []OneOfPair
	for _, k := range d {
		v, ok := k.(ProductType)
		if !ok {
			continue
		}
		for field, info := range v.Fields {
			ot, ok := info.Types.(OneOfType)
			if !ok {
				continue
			}
			name := `OneOf` + strcase.ToCamel(k.Name()) + strcase.ToCamel(field)
			for _, v := range ot.Types {
				if v.Type == s {
					ret = append(ret, OneOfPair{name, ot})
				}
			}
		}
		if c := v.Children; c != nil {
			ot, ok := c.Types.(OneOfType)
			if !ok {
				continue
			}
			name := `OneOf` + strcase.ToCamel(k.Name())
			for _, v := range ot.Types {
				if v.Type == s {
					ret = append(ret, OneOfPair{name, ot})
				}
			}
		}
	}
	for _, v := range d.FindSupertypesFor(s) {
		ret = append(ret, d.OneOfTypesIncluding(v.Name())...)
	}
	return ret
}

func (d DataTypes) FindSupertypesFor(s string) []SumType {
	var ret []SumType
	for _, v := range d {
		vv, ok := v.(SumType)
		if !ok {
			continue
		}
		for _, t := range vv.Subtypes {
			if t.Type == s {
				ret = append(ret, vv)
			}
		}
	}
	return ret
}

type CommonInfo struct {
	DName  string
	DNamed bool
}

func (c CommonInfo) Name() string { return c.DName }
func (c CommonInfo) Named() bool  { return c.DNamed }

type SumType struct {
	CommonInfo
	Subtypes []Type
}

type ProductType struct {
	CommonInfo
	Children *Field
	Fields   map[string]Field
}

type LeafType struct {
	CommonInfo
}

type JSONField struct {
	Required bool   `json:"required"`
	Multiple bool   `json:"multiple"`
	Types    []Type `json:"types"`
}

func (j *JSONField) Into() *Field {
	if j == nil {
		return nil
	}
	UnmarshalTypes := func(t []Type) Types {
		if len(t) == 0 {
			panic("bad types")
		}
		if len(t) == 1 {
			return SingleType{t[0]}
		}
		isNamed := t[0].Named
		for _, st := range t {
			if st.Named != isNamed {
				isNamed = true
			}
		}
		return OneOfType{t, isNamed}
	}
	return &Field{j.Required, j.Multiple, UnmarshalTypes(j.Types)}
}

type Field struct {
	Required bool
	Multiple bool
	Types    Types
}

type Namedness int

const (
	Named Namedness = iota
	Nameless
	NameHeterogenuous
)

type Types interface {
	isTypes()
	Named() bool
}

type SingleType struct {
	Type Type
}

func (SingleType) isTypes() {}

func (t SingleType) Named() bool {
	return t.Type.Named
}

type OneOfType struct {
	Types  []Type
	DNamed bool
}

func (OneOfType) isTypes() {}

func (t OneOfType) Named() bool {
	return t.DNamed
}

type Type struct {
	Type  string `json:"type"`
	Named bool   `json:"named"`
}
