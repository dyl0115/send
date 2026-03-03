package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Config struct {
	GmailFrom     string `json:"gmail_from"`
	GmailPassword string `json:"gmail_password"`
}

func getConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "send", "config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config 파일 없음. 'send config set' 으로 설정해줘!")
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}
	// 폴더 없으면 자동 생성
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600) // 0600 = 본인만 읽기/쓰기
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "send 설정 관리",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "설정값 저장",
	Run: func(cmd *cobra.Command, args []string) {
		// 기존 config 불러오기 (없으면 새로 생성)
		cfg, err := loadConfig()
		if err != nil {
			cfg = &Config{}
		}

		if v, _ := cmd.Flags().GetString("gmail-from"); v != "" {
			cfg.GmailFrom = v
		}
		if v, _ := cmd.Flags().GetString("gmail-password"); v != "" {
			cfg.GmailPassword = v
		}

		if err := saveConfig(cfg); err != nil {
			fmt.Println("❌ 저장 실패:", err)
			os.Exit(1)
		}
		fmt.Println("✅ 설정 저장 완료!")
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "현재 설정 확인",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
		fmt.Println("📧 gmail_from    :", cfg.GmailFrom)
		fmt.Println("🔑 gmail_password:", "****"+cfg.GmailPassword[len(cfg.GmailPassword)-4:])
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)

	configSetCmd.Flags().String("gmail-from", "", "Gmail 주소")
	configSetCmd.Flags().String("gmail-password", "", "Gmail 앱 비밀번호")
}
