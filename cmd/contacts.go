package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// resolveRecipient: 별칭이면 이메일로 변환, 아니면 그대로 반환
func resolveRecipient(input string) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return input, nil // config 없으면 그냥 그대로 사용
	}
	if email, ok := cfg.Contacts[input]; ok {
		return email, nil
	}
	return input, nil
}

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "주소록 관리",
}

var contactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "주소록 목록 출력",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
		if len(cfg.Contacts) == 0 {
			fmt.Println("📭 주소록이 비어있어요. 'send contacts add'로 추가해보세요!")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "별칭\t이메일")
		fmt.Fprintln(w, "────\t──────")
		for alias, email := range cfg.Contacts {
			fmt.Fprintf(w, "%s\t%s\n", alias, email)
		}
		w.Flush()
	},
}

var contactsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "주소록에 연락처 추가",
	Run: func(cmd *cobra.Command, args []string) {
		alias, _ := cmd.Flags().GetString("alias")
		email, _ := cmd.Flags().GetString("email")

		cfg, err := loadConfig()
		if err != nil {
			cfg = &Config{Contacts: map[string]string{}}
		}
		if cfg.Contacts == nil {
			cfg.Contacts = map[string]string{}
		}
		cfg.Contacts[alias] = email
		if err := saveConfig(cfg); err != nil {
			fmt.Println("❌ 저장 실패:", err)
			os.Exit(1)
		}
		fmt.Printf("✅ 주소록 추가 완료! [%s] → %s\n", alias, email)
	},
}

var contactsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "주소록에서 연락처 삭제",
	Run: func(cmd *cobra.Command, args []string) {
		alias, _ := cmd.Flags().GetString("alias")

		cfg, err := loadConfig()
		if err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
		if _, ok := cfg.Contacts[alias]; !ok {
			fmt.Printf("❌ [%s] 별칭을 찾을 수 없어요.\n", alias)
			os.Exit(1)
		}
		delete(cfg.Contacts, alias)
		if err := saveConfig(cfg); err != nil {
			fmt.Println("❌ 저장 실패:", err)
			os.Exit(1)
		}
		fmt.Printf("✅ [%s] 삭제 완료!\n", alias)
	},
}

func init() {
	rootCmd.AddCommand(contactsCmd)
	contactsCmd.AddCommand(contactsListCmd)
	contactsCmd.AddCommand(contactsAddCmd)
	contactsCmd.AddCommand(contactsRemoveCmd)

	contactsAddCmd.Flags().String("alias", "", "별칭 (예: mom, me-naver)")
	contactsAddCmd.Flags().String("email", "", "이메일 주소")
	contactsAddCmd.MarkFlagRequired("alias")
	contactsAddCmd.MarkFlagRequired("email")

	contactsRemoveCmd.Flags().String("alias", "", "삭제할 별칭")
	contactsRemoveCmd.MarkFlagRequired("alias")
}
