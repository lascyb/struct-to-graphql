package core

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/lascyb/tagkit"
)

// TypeNamer 由结构体实现（值接收者或指针接收者均可），声明其对应的 GraphQL Schema 类型名，
// 用于 Fragment 的类型条件（fragment X on <Type>）以及 union 分支（... on <Type>）。
// 返回空字符串（或全空白）视为未声明。
type TypeNamer interface {
	GraphQLType() string
}

// FragmentNamer 由结构体实现（值接收者或指针接收者均可），声明其生成 Fragment 时使用的名称
// （fragment <Name> on ... 与 ...<Name>）。返回空字符串（或全空白）视为未声明。
type FragmentNamer interface {
	GraphQLFragment() string
}

var (
	typeNamerType     = reflect.TypeOf((*TypeNamer)(nil)).Elem()
	fragmentNamerType = reflect.TypeOf((*FragmentNamer)(nil)).Elem()
)

const (
	typeFlag     = "type"     // 字段 tag：指定 GraphQL 类型名（union 分支的 ... on，及复用结构体 Fragment 的类型条件）
	fragmentFlag = "fragment" // 字段 tag：指定复用结构体生成 Fragment 时的名称
)

// isGraphQLName 判断 s 是否为合法的 GraphQL Name：[_A-Za-z][_0-9A-Za-z]*
func isGraphQLName(s string) bool {
	return directiveNameRe.MatchString(s)
}

// validateTypeName 校验 GraphQL 类型名
func validateTypeName(name, source string) error {
	if !isGraphQLName(name) {
		return fmt.Errorf("%s: %q is not a valid GraphQL type name (must match [_A-Za-z][_0-9A-Za-z]*)", source, name)
	}
	return nil
}

// validateFragmentName 校验 Fragment 名称；按 GraphQL 规范 Fragment 名不能是 on
func validateFragmentName(name, source string) error {
	if !isGraphQLName(name) || name == "on" {
		return fmt.Errorf("%s: %q is not a valid GraphQL fragment name (must match [_A-Za-z][_0-9A-Za-z]* and cannot be \"on\")", source, name)
	}
	return nil
}

// declaredNames 读取结构体通过 TypeNamer / FragmentNamer 声明的类型名与 Fragment 名，并校验合法性。
// 通过 reflect.New 创建 *T 调用，因此值接收者与指针接收者的方法都能识别，且接收者不会是 nil。
func declaredNames(typ reflect.Type) (typeName, fragmentName string, err error) {
	ptr := reflect.PointerTo(typ)
	if ptr.Implements(typeNamerType) {
		typeName = strings.TrimSpace(reflect.New(typ).Interface().(TypeNamer).GraphQLType())
		if typeName != "" {
			if err := validateTypeName(typeName, typ.String()+".GraphQLType()"); err != nil {
				return "", "", fmt.Errorf("graphql: %w", err)
			}
		}
	}
	if ptr.Implements(fragmentNamerType) {
		fragmentName = strings.TrimSpace(reflect.New(typ).Interface().(FragmentNamer).GraphQLFragment())
		if fragmentName != "" {
			if err := validateFragmentName(fragmentName, typ.String()+".GraphQLFragment()"); err != nil {
				return "", "", fmt.Errorf("graphql: %w", err)
			}
		}
	}
	return typeName, fragmentName, nil
}

// nameFlag 读取字段 tag 中 type=X / fragment=X 的值（去除首尾空白）；未设置或为空时返回 ""
func nameFlag(tv *tagkit.TagValue, flag string) (string, error) {
	if tv == nil {
		return "", nil
	}
	f, ok := tv.Flag(flag)
	if !ok || !f.HasValue() {
		return "", nil
	}
	s, ok := f.Value.(string)
	if !ok {
		return "", fmt.Errorf("%s flag must be a name, got %v", flag, f.Value)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if flag == fragmentFlag {
		return s, validateFragmentName(s, "fragment flag")
	}
	return s, validateTypeName(s, "type flag")
}

// goTypeName 返回 Go 类型名，泛型类型去掉类型参数部分（如 Page[Item] → Page）
func goTypeName(typ reflect.Type) string {
	name, _, _ := strings.Cut(typ.Name(), "[")
	return name
}

// fragmentRef 一次「字段 → 复用结构体」的引用，用于确定 Fragment 的名称与类型条件
type fragmentRef struct {
	where    string // 引用位置，如 "Query.Author"
	typ      string // 字段 tag 中的 type=X
	fragment string // 字段 tag 中的 fragment=X
	union    bool   // 是否为 union 分支
}

// prepareFragments 在构建前遍历整棵类型树，为每个将要生成的 Fragment 确定类型条件与显式名称：
//
// 类型条件（on <Type>）：结构体的 GraphQLType() 优先，此时非 union 字段上的 type=X 必须与之一致
// （union 分支的 type=X 只决定 ... on，可以不同，如 GraphQLType() 返回接口名）；未实现时取所有引用字段上
// 一致的 type=X，出现不同的值时报错；都没有时使用 Go 类型名。
//
// 名称：GraphQLFragment() 优先，其次是所有引用字段上一致的 fragment=X，出现不同的值时报错；
// 显式名称在构建前预先登记，两个类型声明同一名称时报错，自动生成的名称会避开它们。
func (g *Builder) prepareFragments(root *TypeParser) error {
	refs := make(map[*TypeParser][]fragmentRef)
	var order []*TypeParser
	visited := make(map[*TypeParser]bool)
	var walk func(tp *TypeParser)
	walk = func(tp *TypeParser) {
		if visited[tp] {
			return
		}
		visited[tp] = true
		for _, f := range tp.Fields {
			child := f.TypeParser
			if child == nil {
				continue
			}
			// 只有普通字段与 union 分支会生成 Fragment；平铺用的匿名嵌入只会引用已生成的 Fragment
			if !f.Inline || tp.Union {
				if _, ok := refs[child]; !ok {
					order = append(order, child)
				}
				refs[child] = append(refs[child], fragmentRef{
					where:    describeType(tp.source) + "." + f.source.Name,
					typ:      f.typeFlag,
					fragment: f.fragmentFlag,
					union:    tp.Union,
				})
			}
			walk(child)
		}
	}
	walk(root)

	for _, tp := range order {
		if !g.asFragment(tp) {
			continue
		}
		typ, err := resolveFragmentType(tp, refs[tp])
		if err != nil {
			return err
		}
		g.fragmentTypes[tp.source] = typ
		name, err := resolveFragmentName(tp, refs[tp])
		if err != nil {
			return err
		}
		if name == "" {
			continue
		}
		if owner, ok := g.fragmentNames[name]; ok {
			return fmt.Errorf("graphql: fragment name %q is declared by both %s and %s; fragment names must be unique", name, owner, tp.source)
		}
		g.fragmentNames[name] = tp.source
		g.explicitNames[tp.source] = name
	}
	return nil
}

func resolveFragmentType(tp *TypeParser, refs []fragmentRef) (string, error) {
	if tp.graphQLType != "" {
		for _, r := range refs {
			if !r.union && r.typ != "" && r.typ != tp.graphQLType {
				return "", fmt.Errorf("graphql: field %s sets type=%s, which conflicts with %s.GraphQLType() = %q", r.where, r.typ, tp.source, tp.graphQLType)
			}
		}
		return tp.graphQLType, nil
	}
	typ, err := unanimous(tp, refs, typeFlag, func(r fragmentRef) string { return r.typ })
	if err != nil {
		return "", fmt.Errorf("%w; implement GraphQLType() on the struct or use WithFragment(false)", err)
	}
	if typ != "" {
		return typ, nil
	}
	return goTypeName(tp.source), nil
}

func resolveFragmentName(tp *TypeParser, refs []fragmentRef) (string, error) {
	if tp.fragmentName != "" {
		return tp.fragmentName, nil
	}
	name, err := unanimous(tp, refs, fragmentFlag, func(r fragmentRef) string { return r.fragment })
	if err != nil {
		return "", fmt.Errorf("%w; implement GraphQLFragment() on the struct or use WithFragment(false)", err)
	}
	return name, nil
}

// unanimous 返回所有引用中非空且一致的取值；出现两个不同的值时报错
func unanimous(tp *TypeParser, refs []fragmentRef, flag string, get func(fragmentRef) string) (string, error) {
	var value, where string
	for _, r := range refs {
		v := get(r)
		if v == "" {
			continue
		}
		if value == "" {
			value, where = v, r.where
			continue
		}
		if v != value {
			return "", fmt.Errorf("graphql: conflicting %s= flags on reused struct %s: field %s has %s=%s, field %s has %s=%s, but only one fragment is generated",
				flag, tp.source, where, flag, value, r.where, flag, v)
		}
	}
	return value, nil
}

func describeType(typ reflect.Type) string {
	if typ.Name() == "" {
		return "(anonymous struct)"
	}
	return typ.Name()
}
