package cmd

import (
	"autocert/internal/backup"
	"autocert/internal/logger"
	"fmt"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出证书和配置",
	Long: `导出证书、私钥和相关配置到压缩包，便于迁移到其他机器。

示例:
  autocert export --output certs.tar.gz
  autocert export --output certs.tar.gz --domain example.com
  autocert export --output certs.zip --format zip`,
	RunE: runExport,
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "导入证书和配置",
	Long: `从压缩包导入证书、私钥和相关配置。

示例:
  autocert import certs.tar.gz
  autocert import certs.zip --restore-schedule`,
	RunE: runImport,
}

var (
	outputFile      string
	exportFormat    string
	exportDomain    string
	restoreSchedule bool
)

func init() {
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)

	// export 命令参数
	exportCmd.Flags().StringVarP(&outputFile, "output", "o", "autocert-backup.tar.gz", "输出文件路径")
	exportCmd.Flags().StringVar(&exportFormat, "format", "tar.gz", "导出格式 (tar.gz, zip)")
	exportCmd.Flags().StringVar(&exportDomain, "domain", "", "只导出指定域名的证书（可选）")

	// import 命令参数
	importCmd.Flags().BoolVar(&restoreSchedule, "restore-schedule", true, "是否恢复定时任务")
}

func runExport(cmd *cobra.Command, args []string) error {
	logger.Info("开始导出证书和配置", "output", outputFile)

	// 创建备份管理器
	backupManager := backup.NewManager()

	// 设置导出选项
	options := &backup.ExportOptions{
		OutputFile: outputFile,
		Format:     exportFormat,
		Domain:     exportDomain,
	}

	// 执行导出
	if err := backupManager.Export(options); err != nil {
		logger.Error("导出失败", "error", err)
		return fmt.Errorf("导出失败: %w", err)
	}

	logger.Info("导出完成", "output", outputFile)
	fmt.Printf("✓ 证书和配置已导出到: %s\n", outputFile)

	return nil
}

func runImport(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请指定要导入的文件")
	}

	inputFile := args[0]
	logger.Info("开始导入证书和配置", "input", inputFile)

	// 创建备份管理器
	backupManager := backup.NewManager()

	// 设置导入选项
	options := &backup.ImportOptions{
		InputFile:       inputFile,
		RestoreSchedule: restoreSchedule,
	}

	// 执行导入
	if err := backupManager.Import(options); err != nil {
		logger.Error("导入失败", "error", err)
		return fmt.Errorf("导入失败: %w", err)
	}

	logger.Info("导入完成", "input", inputFile)
	fmt.Printf("✓ 证书和配置已从 %s 导入\n", inputFile)

	return nil
}
