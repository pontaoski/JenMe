package main

import (
	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	gotreesitter "github.com/smacker/go-tree-sitter"
)

var nodeType = jen.Op("*").Qual("github.com/smacker/go-tree-sitter", "Node")

func GenerateMultiMkFunc(sig *jen.Statement, subtypes []Type) {
	sig.BlockFunc(func(g *jen.Group) {
		for _, v := range subtypes {
			if !v.Named {
				continue
			}
			g.If(jen.Id(`n`).Dot(`Type`).Call().Op(`==`).Lit(v.Type)).BlockFunc(func(g *jen.Group) {
				g.Return(jen.Id(`mk` + strcase.ToCamel(v.Type)).Call(jen.Id(`n`)))
			})
		}
		g.Panic(jen.Lit(`Unhandled case`).Op(`+`).Id(`n`).Dot(`Type`).Call())
	})
}

func GenerateSumType(f *jen.File, k SumType, ns DataTypes, makeSig *jen.Statement) {
	f.Type().Id(strcase.ToCamel(k.Name())).InterfaceFunc(func(g *jen.Group) {
		for _, st := range ns.FindSupertypesFor(k.Name()) {
			g.Id(strcase.ToCamel(st.Name()))
		}
		for _, ot := range ns.OneOfTypesIncluding(k.Name()) {
			g.Id(ot.Name)
		}
		g.Id(`is` + strcase.ToCamel(k.Name())).Params()
		g.Id(`GetNode`).Params().Add(nodeType)
	})
	for _, v := range k.Subtypes {
		if !v.Named {
			continue
		}
		if _, ok := ns.FindType(v.Type).(SumType); ok {
			continue
		}
		for _, st := range ns.FindSupertypesFor(k.Name()) {
			f.Func().Params(jen.Id(strcase.ToCamel(v.Type))).Id(`is` + strcase.ToCamel(st.Name())).Params().Block()
		}
		f.Func().Params(jen.Id(strcase.ToCamel(v.Type))).Id(`is` + strcase.ToCamel(k.Name())).Params().Block()
	}
	GenerateMultiMkFunc(makeSig, k.Subtypes)
}

func GenerateProductTypeMultiGetter() {

}

func GenerateProductTypeFieldGetter(f *jen.File, k ProductType, ns DataTypes, sig *jen.Statement, name string, info Field, isField bool) {
	if info.Multiple {
		sig.Index()
	}

	switch v := info.Types.(type) {
	case SingleType:
		sig.Id(strcase.ToCamel(v.Type.Type))
	case OneOfType:
		name := `OneOf` + strcase.ToCamel(k.Name()) + strcase.ToCamel(name)
		if !isField {
			name = `OneOf` + strcase.ToCamel(k.Name())
		}
		sig.Id(name)
		f.Type().Id(name).InterfaceFunc(func(g *jen.Group) {
			g.Id(`is` + name).Params()
			g.Id(`GetNode`).Params().Add(nodeType)
		})
		sig := f.Func().Id(`mk` + name).Params(jen.Id(`n`).Add(nodeType)).Id(name)
		GenerateMultiMkFunc(sig, v.Types)
	}

	sig.BlockFunc(func(fn *jen.Group) {
		switch v := info.Types.(type) {
		case SingleType:
			if !info.Multiple {
				if isField {
					fn.Return(jen.Id(`mk` + strcase.ToCamel(v.Type.Type)).Call(jen.Id(`n`).Dot(`ChildByFieldName`).Call(jen.Lit(name))))
				} else {
					r := fn.Return(jen.Id(`mk` + strcase.ToCamel(v.Type.Type)))
					r.Call(jen.Id(`n`).Dot(`NamedChild`).Call(jen.Lit(0)))
				}
			} else {
				if isField {
					fn.Id(`ret`).Op(`:=`).Call(jen.Index().Id(strcase.ToCamel(name)).Values())

					fn.For(
						jen.Id(`i`).Op(`:=`).Lit(0),
						jen.Uint32().Call(jen.Id(`i`)).Op(`<`).CustomFunc(jen.Options{}, func(g *jen.Group) {
							g.Id(`n`).Dot(`ChildCount`).Call()
						}),
						jen.Id(`i`).Op(`++`),
					).BlockFunc(func(g *jen.Group) {
						g.If(jen.Id(`n`).Dot(`FieldNameForChild`).Call(jen.Id(`i`)).Op(`!=`).Lit(name)).Block(jen.Continue())
						g.Id(`ret`).Op(`=`).AppendFunc(func(g *jen.Group) {
							g.Id(`ret`)
							r := g.Id(`mk` + strcase.ToCamel(name))
							r.Call(jen.Id(`n`).Dot(`NamedChild`).Call(jen.Id(`i`)))
						})
					})
					fn.Return(jen.Id(`ret`))
				} else {
					name := strcase.ToCamel(v.Type.Type)
					fn.Id(`ret`).Op(`:=`).Call(jen.Index().Id(name).Values())
					fn.For(
						jen.Id(`i`).Op(`:=`).Lit(0),
						jen.Uint32().Call(jen.Id(`i`)).Op(`<`).CustomFunc(jen.Options{}, func(g *jen.Group) {
							g.Id(`n`).Dot(`NamedChildCount`).Call()
						}),
						jen.Id(`i`).Op(`++`),
					).BlockFunc(func(g *jen.Group) {
						g.Id(`ret`).Op(`=`).AppendFunc(func(g *jen.Group) {
							g.Id(`ret`)
							r := g.Id(`mk` + strcase.ToCamel(v.Type.Type))
							r.Call(jen.Id(`n`).Dot(`NamedChild`).Call(jen.Id(`i`)))
						})
					})
					fn.Return(jen.Id(`ret`))
				}
			}
			_ = v
		case OneOfType:
			name := `OneOf` + strcase.ToCamel(k.Name()) + strcase.ToCamel(name)
			if !isField {
				name = `OneOf` + strcase.ToCamel(k.Name())
			}

			if !info.Multiple {
				if isField {
					fn.Return(jen.Id(`mk` + name).Call(jen.Id(`n`).Dot(`ChildByFieldName`).Call(jen.Lit(name))))
				} else {
					fn.Return(jen.Id(`mk` + name).Call(jen.Id(`n`).Dot(`NamedChild`).Call(jen.Lit(0))))
				}
			} else {
				if isField {
					fn.Id(`ret`).Op(`:=`).Call(jen.Index().Id(name).Values())

					fn.For(
						jen.Id(`i`).Op(`:=`).Lit(0),
						jen.Uint32().Call(jen.Id(`i`)).Op(`<`).CustomFunc(jen.Options{}, func(g *jen.Group) {
							g.Id(`n`).Dot(`ChildCount`).Call()
						}),
						jen.Id(`i`).Op(`++`),
					).BlockFunc(func(g *jen.Group) {
						// g.If(jen.Id(`n`).Dot(`FieldNameForChild`).Call(jen.Id(`i`)).Op(`!=`).Lit(name)).Block(jen.Continue())
						g.Id(`ret`).Op(`=`).AppendFunc(func(g *jen.Group) {
							g.Id(`ret`)
							r := g.Id(`mk` + name)
							r.Call(jen.Id(`n`).Dot(`NamedChild`).Call(jen.Id(`i`)))
						})
					})
					fn.Return(jen.Id(`ret`))
				} else {
					fn.Id(`ret`).Op(`:=`).Call(jen.Index().Id(name).Values())
					fn.For(
						jen.Id(`i`).Op(`:=`).Lit(0),
						jen.Uint32().Call(jen.Id(`i`)).Op(`<`).CustomFunc(jen.Options{}, func(g *jen.Group) {
							g.Id(`n`).Dot(`NamedChildCount`).Call()
						}),
						jen.Id(`i`).Op(`++`),
					).BlockFunc(func(g *jen.Group) {
						g.Id(`ret`).Op(`=`).AppendFunc(func(g *jen.Group) {
							g.Id(`ret`)
							r := g.Id(`mk` + name)
							r.Call(jen.Id(`n`).Dot(`NamedChild`).Call(jen.Id(`i`)))
						})
					})
					fn.Return(jen.Id(`ret`))
				}
			}
		}
	})
}

func GenerateProductType(f *jen.File, k ProductType, ns DataTypes, makeSig *jen.Statement) {
	f.Type().Id(strcase.ToCamel(k.Name())).StructFunc(func(g *jen.Group) {
		g.Add(nodeType)
	})
	makeSig.BlockFunc(func(g *jen.Group) {
		g.Return(jen.Id(strcase.ToCamel(k.Name())).Values(jen.Id(`n`)))
	})

	for field, info := range k.Fields {
		if !info.Types.Named() {
			continue
		}
		sig := f.Func().Parens(jen.Id("n").Op("*").Id(strcase.ToCamel(k.Name()))).Id(strcase.ToCamel(field)).Params()
		GenerateProductTypeFieldGetter(f, k, ns, sig, field, info, true)
	}
	if c := k.Children; c != nil {
		sig := f.Func().Parens(jen.Id("n").Op("*").Id(strcase.ToCamel(k.Name())))
		if c.Multiple {
			sig = sig.Id(strcase.ToCamel(`Children`)).Params()
		} else {
			sig = sig.Id(strcase.ToCamel(`Child`)).Params()
		}

		GenerateProductTypeFieldGetter(f, k, ns, sig, ``, *c, false)
	}
}

func Generate(f *jen.File, n DataType, ns DataTypes) {
	var s *gotreesitter.Node
	_ = s
	if !n.Named() {
		return
	}
	makeSig := f.Func().Id(`mk` + strcase.ToCamel(n.Name())).Params(jen.Id(`n`).Add(nodeType)).Id(strcase.ToCamel(n.Name()))
	_ = makeSig
	switch k := n.(type) {
	case SumType:
		GenerateSumType(f, k, ns, makeSig)
	case ProductType:
		for _, ot := range ns.OneOfTypesIncluding(k.Name()) {
			f.Func().Params(jen.Id(strcase.ToCamel(k.Name()))).Id(`is` + ot.Name).Params().Block()
		}
		f.Func().Params(jen.Id(`n`).Id(strcase.ToCamel(k.Name()))).Id(`GetNode`).Params().Add(nodeType).BlockFunc(func(g *jen.Group) {
			g.Return(jen.Id(`n`).Dot(`Node`))
		})
		GenerateProductType(f, k, ns, makeSig)
	case LeafType:
		for _, ot := range ns.OneOfTypesIncluding(k.Name()) {
			f.Func().Params(jen.Id(strcase.ToCamel(k.Name()))).Id(`is` + ot.Name).Params().Block()
		}
		f.Func().Params(jen.Id(`n`).Id(strcase.ToCamel(k.Name()))).Id(`GetNode`).Params().Add(nodeType).BlockFunc(func(g *jen.Group) {
			g.Return(jen.Id(`n`).Dot(`Node`))
		})
		f.Type().Id(strcase.ToCamel(k.Name())).StructFunc(func(g *jen.Group) {
			g.Add(nodeType)
		})
		makeSig.BlockFunc(func(g *jen.Group) {
			g.Return(jen.Id(strcase.ToCamel(k.Name())).Values(jen.Id(`n`)))
		})
	}
}
