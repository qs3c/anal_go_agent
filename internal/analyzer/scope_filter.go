package analyzer

import (
	"strings"
	"unicode"

	"github.com/user/go-struct-analyzer/internal/parser"
)

// ScopeFilter 用于过滤分析范围
type ScopeFilter struct {
	parser          *parser.Parser
	blacklist       *Blacklist
	projectPackages []string
	moduleName      string
}

// NewScopeFilter 创建范围过滤器
func NewScopeFilter(p *parser.Parser, blacklist *Blacklist) *ScopeFilter {
	sf := &ScopeFilter{
		parser:     p,
		blacklist:  blacklist,
		moduleName: p.GetModuleName(),
	}

	// 收集项目内的所有包
	sf.collectProjectPackages()

	return sf
}

// collectProjectPackages 收集项目内的所有包
func (sf *ScopeFilter) collectProjectPackages() {
	packages := make(map[string]bool)

	for _, structInfo := range sf.parser.GetAllStructs() {
		packages[structInfo.Package] = true
	}

	for pkg := range packages {
		sf.projectPackages = append(sf.projectPackages, pkg)
	}
}

// ShouldAnalyze 判断类型是否应该被分析
func (sf *ScopeFilter) ShouldAnalyze(typeName string) bool {
	// 0. 跳过空类型名
	if typeName == "" {
		return false
	}

	// 1. 清理类型名
	typeName = strings.TrimPrefix(typeName, "*")
	typeName = strings.TrimPrefix(typeName, "[]")

	// 去掉 map 的类型
	if strings.HasPrefix(typeName, "map[") {
		return false
	}

	// 2. 跳过基础类型
	if isBuiltinType(typeName) {
		return false
	}

	// 3. 跳过标准库类型
	if isStandardLibrary(typeName) {
		return false
	}

	// 4. 检查黑名单
	if sf.blacklist != nil && sf.blacklist.IsBlocked(typeName) {
		return false
	}

	// 5. 验证类型名格式：Go 导出类型必须首字母大写
	// 这可以过滤掉被误识别的变量名（通常小写开头）
	baseTypeName := typeName
	if idx := strings.LastIndex(typeName, "."); idx != -1 {
		baseTypeName = typeName[idx+1:]
	}
	if len(baseTypeName) > 0 && !unicode.IsUpper(rune(baseTypeName[0])) {
		return false
	}

	// 6. 检查是否为项目内部类型
	return sf.isInternalType(typeName)
}

// isInternalType 判断是否为项目内部类型
func (sf *ScopeFilter) isInternalType(typeName string) bool {
	// 如果类型名不包含点，可能是当前包的类型
	if !strings.Contains(typeName, ".") {
		// 检查是否在已知结构体中
		if sf.parser.GetStruct(typeName) != nil {
			return true
		}
		// 如果在已知包中，也认为是内部类型
		for _, pkg := range sf.projectPackages {
			if pkg == typeName {
				return true
			}
		}
	}

	// 检查是否以项目模块名开头
	if sf.moduleName != "" && strings.HasPrefix(typeName, sf.moduleName) {
		return true
	}

	// 检查是否在项目包列表中
	parts := strings.Split(typeName, ".")
	if len(parts) > 0 {
		pkgName := parts[0]
		for _, pkg := range sf.projectPackages {
			if pkg == pkgName {
				return true
			}
		}
	}

	// 如果不包含 "/" 且解析器能找到该结构体，认为是内部类型
	if !strings.Contains(typeName, "/") && sf.parser.GetStruct(typeName) != nil {
		return true
	}

	return false
}

// isBuiltinType 判断是否为内置类型
func isBuiltinType(typeName string) bool {
	builtins := map[string]bool{
		"bool":       true,
		"string":     true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
		"byte":       true,
		"rune":       true,
		"float32":    true,
		"float64":    true,
		"complex64":  true,
		"complex128": true,
		"error":      true,
		"any":        true,
		"interface{}" : true,
	}

	return builtins[typeName]
}

// isStandardLibrary 判断是否为标准库类型
func isStandardLibrary(typeName string) bool {
	stdLibs := []string{
		"context",
		"net",
		"io",
		"os",
		"fmt",
		"time",
		"sync",
		"errors",
		"strings",
		"bytes",
		"http",
		"json",
		"xml",
		"sql",
		"crypto",
		"encoding",
		"bufio",
		"log",
		"path",
		"filepath",
		"regexp",
		"sort",
		"strconv",
		"testing",
		"reflect",
		"runtime",
		"unsafe",
		"math",
		"flag",
		"html",
		"image",
		"mime",
		"template",
		"text",
		"unicode",
		"archive",
		"compress",
		"container",
		"database",
		"debug",
		"embed",
		"expvar",
		"go",
		"hash",
		"index",
		"plugin",
		"syscall",
	}

	// 检查是否以标准库包名开头
	parts := strings.Split(typeName, ".")
	if len(parts) > 0 {
		basePkg := parts[0]
		for _, std := range stdLibs {
			if basePkg == std {
				return true
			}
		}
	}

	return false
}
