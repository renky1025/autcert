package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// 这些变量将在编译时通过 ldflags 设置
	version    = "dev"
	buildTime  = "unknown"
	commitHash = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  `显示 AutoCert 的版本信息，包括版本号、构建时间和 Git 提交哈希。`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// SetVersionInfo 设置版本信息（由 main 函数调用）
func SetVersionInfo(v, bt, ch string) {
	version = v
	buildTime = bt
	commitHash = ch
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("AutoCert Version: %s\n", version)
	fmt.Printf("Build Time: %s\n", buildTime)
	fmt.Printf("Git Commit: %s\n", commitHash)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
