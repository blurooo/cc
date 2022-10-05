// Package option 提供基于位运算的附加选项能力
package option

// Option 选项
type Option int

// Has 是否存在某个选项
func Has(srcOptions []Option, target Option) bool {
	var useOption Option
	if len(srcOptions) != 0 {
		useOption = target
	}
	for _, option := range srcOptions {
		useOption = useOption & option
	}
	return useOption == target
}
