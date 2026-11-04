package core

import (
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// fragmentName 为类型分配一个合法且唯一的 Fragment 名称。
//
// 规则：
//  1. 默认使用 Go 类型名（不含包名），如 UserInfo；
//  2. 泛型类型将类型参数的短名拼接在后面，如 Page[pkg.Item] → Page_Item，Pair[string,int] → Pair_String_Int；
//  3. 名称中不合法的字符替换为下划线；
//  4. 不同类型得到相同名称时（如不同包下的同名类型），后出现的类型追加包名后缀，如 User_models；
//     仍然冲突则继续追加 _2、_3……。名称按 Fragment 构建完成的顺序分配，因此结果是确定的。
func (g *Builder) fragmentName(typ reflect.Type) string {
	base := fragmentBaseName(typ.Name())
	candidates := []string{base}
	if pkg := packageName(typ.PkgPath()); pkg != "" {
		candidates = append(candidates, base+"_"+pkg)
	}
	for _, name := range candidates {
		if g.claimFragmentName(name, typ) {
			return name
		}
	}
	last := candidates[len(candidates)-1]
	for i := 2; ; i++ {
		name := last + "_" + strconv.Itoa(i)
		if g.claimFragmentName(name, typ) {
			return name
		}
	}
}

func (g *Builder) claimFragmentName(name string, typ reflect.Type) bool {
	if owner, ok := g.fragmentNames[name]; ok && owner != typ {
		return false
	}
	g.fragmentNames[name] = typ
	return true
}

// fragmentBaseName 由 reflect.Type.Name() 生成 Fragment 基础名称
func fragmentBaseName(typeName string) string {
	base, args, isGeneric := strings.Cut(typeName, "[")
	parts := []string{upperFirst(base)}
	if isGeneric {
		for _, tok := range strings.FieldsFunc(args, func(r rune) bool {
			return strings.ContainsRune("[]*, {};", r)
		}) {
			// 去掉包路径，只保留短名：github.com/a/b.Item → Item
			if i := strings.LastIndexByte(tok, '.'); i >= 0 {
				tok = tok[i+1:]
			}
			parts = append(parts, upperFirst(tok))
		}
	}
	return sanitizeName(strings.Join(parts, "_"))
}

// packageName 返回包路径的最后一段并清理为合法名称
func packageName(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}
	if i := strings.LastIndexByte(pkgPath, '/'); i >= 0 {
		pkgPath = pkgPath[i+1:]
	}
	return sanitizeName(pkgPath)
}

// sanitizeName 将非 [_0-9A-Za-z] 字符替换为下划线，合并连续下划线，并保证不以数字开头
func sanitizeName(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		ok := r == '_' || (r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)))
		if !ok {
			r = '_'
		}
		if r == '_' {
			if lastUnderscore {
				continue
			}
			lastUnderscore = true
		} else {
			lastUnderscore = false
		}
		b.WriteRune(r)
	}
	name := strings.Trim(b.String(), "_")
	if name == "" {
		return "Fragment"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "F_" + name
	}
	return name
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
