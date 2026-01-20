package main

import (
	"fmt"

	"github.com/user/go-struct-analyzer/internal/analyzer"
	"github.com/user/go-struct-analyzer/internal/parser"
)

func main() {
	p := parser.NewParser(true)
	err := p.ParseProject("./testdata/sample_project")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("=== Interfaces ===")
	for name, iface := range p.GetAllInterfaces() {
		fmt.Printf("Interface: %s (Package: %s)\n", name, iface.Package)
		for _, m := range iface.Methods {
			fmt.Printf("  - %s%s\n", m.Name, m.Signature)
		}
	}

	fmt.Println("\n=== Functions (New*) ===")
	for key, fn := range p.GetAllFunctions() {
		fmt.Printf("Function: %s -> %s\n", key, fn.ReturnType)
	}

	fmt.Println("\n=== Testing Interface Implementation ===")
	blacklist := analyzer.NewBlacklist()
	filter := analyzer.NewScopeFilter(p, blacklist)

	// Check if UserRepositoryInterface passes the filter
	fmt.Printf("ShouldAnalyze(UserRepositoryInterface): %v\n", filter.ShouldAnalyze("UserRepositoryInterface"))

	// Check struct methods
	userRepo := p.GetStruct("UserRepository")
	if userRepo != nil {
		fmt.Printf("\nUserRepository methods:\n")
		for _, m := range userRepo.Methods {
			fmt.Printf("  - %s %s\n", m.Name, m.Signature)
		}
	}
}
