package analyzer

import (
	"os"
	"strings"

	"github.com/user/go-struct-analyzer/internal/types"
	"gopkg.in/yaml.v3"
)

// Blacklist 管理黑名单类型
type Blacklist struct {
	types    map[string]bool
	packages map[string]bool
}

// NewBlacklist 创建黑名单
func NewBlacklist() *Blacklist {
	return &Blacklist{
		types:    make(map[string]bool),
		packages: make(map[string]bool),
	}
}

// LoadFromFile 从文件加载黑名单配置
func (b *Blacklist) LoadFromFile(filePath string) error {
	if filePath == "" {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var config types.BlacklistConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	for _, t := range config.Types {
		b.types[t] = true
	}

	for _, p := range config.Packages {
		b.packages[p] = true
	}

	return nil
}

// AddType 添加类型到黑名单
func (b *Blacklist) AddType(typeName string) {
	b.types[typeName] = true
}

// AddPackage 添加包到黑名单
func (b *Blacklist) AddPackage(pkgName string) {
	b.packages[pkgName] = true
}

// IsBlocked 检查类型是否在黑名单中
func (b *Blacklist) IsBlocked(typeName string) bool {
	// 清理类型名
	typeName = strings.TrimPrefix(typeName, "*")
	typeName = strings.TrimPrefix(typeName, "[]")

	// 检查类型是否在黑名单
	if b.types[typeName] {
		return true
	}

	// 检查是否包含包名前缀
	parts := strings.Split(typeName, ".")
	if len(parts) > 1 {
		pkgName := parts[0]
		if b.packages[pkgName] {
			return true
		}
		// 检查类型名（不带包前缀）
		shortName := parts[len(parts)-1]
		if b.types[shortName] {
			return true
		}
	}

	// 检查完整路径是否匹配包黑名单
	for pkg := range b.packages {
		if strings.HasPrefix(typeName, pkg) {
			return true
		}
	}

	return false
}

// GetBlockedTypes 获取所有被阻止的类型
func (b *Blacklist) GetBlockedTypes() []string {
	var result []string
	for t := range b.types {
		result = append(result, t)
	}
	return result
}

// GetBlockedPackages 获取所有被阻止的包
func (b *Blacklist) GetBlockedPackages() []string {
	var result []string
	for p := range b.packages {
		result = append(result, p)
	}
	return result
}
