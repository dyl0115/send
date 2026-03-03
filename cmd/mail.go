package cmd

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"

	"github.com/spf13/cobra"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "이메일 전송",
	Run: func(cmd *cobra.Command, args []string) {
		to, _ := cmd.Flags().GetString("to")
		subject, _ := cmd.Flags().GetString("subject")
		body, _ := cmd.Flags().GetString("body")

		err := sendMail(to, subject, body)
		if err != nil {
			fmt.Println("❌ 전송 실패:", err)
			os.Exit(1)
		}
		fmt.Println("✅ 메일 전송 완료!")
	},
}

func init() {
	rootCmd.AddCommand(mailCmd)
	mailCmd.Flags().String("to", "", "수신자 이메일")
	mailCmd.Flags().String("subject", "알림", "제목")
	mailCmd.Flags().String("body", "", "내용")
	mailCmd.MarkFlagRequired("to")
	mailCmd.MarkFlagRequired("body")
}

func sendMail(to, subject, body string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	from := cfg.GmailFrom
	password := cfg.GmailPassword
	msg := "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + body

	auth := smtp.PlainAuth("", from, password, "smtp.gmail.com")

	tlsConfig := &tls.Config{ServerName: "smtp.gmail.com"}
	conn, err := tls.Dial("tcp", "smtp.gmail.com:465", tlsConfig)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, "smtp.gmail.com")
	if err != nil {
		return err
	}
	if err = client.Auth(auth); err != nil {
		return err
	}
	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	w.Close()
	client.Quit()
	return nil
}
