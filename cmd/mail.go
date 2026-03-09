package cmd

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "이메일 전송",
	Run: func(cmd *cobra.Command, args []string) {
		to, _ := cmd.Flags().GetString("to")
		subject, _ := cmd.Flags().GetString("subject")
		body, _ := cmd.Flags().GetString("body")
		files, _ := cmd.Flags().GetStringArray("file")

		// 별칭 → 이메일 resolve
		resolved, err := resolveRecipient(to)
		if err != nil {
			fmt.Println("❌ 주소록 오류:", err)
			os.Exit(1)
		}
		if !strings.Contains(resolved, "@") {
			fmt.Printf("❌ [%s] 는 유효한 이메일도 아니고 주소록에도 없어요.\n", to)
			fmt.Println("   'send contacts list' 로 등록된 별칭을 확인해보세요.")
			os.Exit(1)
		}
		if resolved != to {
			fmt.Printf("📒 [%s] → %s\n", to, resolved)
		}

		// 첨부파일 존재 여부 확인
		for _, f := range files {
			if _, err := os.Stat(f); err != nil {
				fmt.Printf("❌ 첨부파일을 찾을 수 없어요: %s\n", f)
				os.Exit(1)
			}
		}

		if err := sendMail(resolved, subject, body, files); err != nil {
			fmt.Println("❌ 전송 실패:", err)
			os.Exit(1)
		}

		if len(files) > 0 {
			fmt.Printf("✅ 메일 전송 완료! (첨부파일 %d개)\n", len(files))
		} else {
			fmt.Println("✅ 메일 전송 완료!")
		}
	},
}

func init() {
	rootCmd.AddCommand(mailCmd)
	mailCmd.Flags().String("to", "", "수신자 이메일 또는 주소록 별칭")
	mailCmd.Flags().String("subject", "알림", "제목")
	mailCmd.Flags().String("body", "", "내용")
	mailCmd.Flags().StringArray("file", []string{}, "첨부파일 경로 (여러 개 가능: --file a.txt --file b.pdf)")
	mailCmd.MarkFlagRequired("to")
	mailCmd.MarkFlagRequired("body")
}

func buildMessage(from, to, subject, body string, files []string) ([]byte, error) {
	var buf bytes.Buffer

	if len(files) == 0 {
		// 첨부파일 없으면 단순 텍스트 메일
		buf.WriteString("From: " + from + "\r\n")
		buf.WriteString("To: " + to + "\r\n")
		buf.WriteString("Subject: " + subject + "\r\n")
		buf.WriteString("MIME-Version: 1.0\r\n")
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(body)
		return buf.Bytes(), nil
	}

	// 첨부파일 있으면 multipart/mixed
	writer := multipart.NewWriter(&buf)

	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/mixed; boundary=\"" + writer.Boundary() + "\"\r\n")
	buf.WriteString("\r\n")

	// 본문 파트
	bodyHeader := textproto.MIMEHeader{}
	bodyHeader.Set("Content-Type", "text/plain; charset=\"UTF-8\"")
	bodyHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	bodyPart, err := writer.CreatePart(bodyHeader)
	if err != nil {
		return nil, err
	}
	bodyPart.Write([]byte(body))

	// 첨부파일 파트들
	for _, filePath := range files {
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("파일 읽기 실패 (%s): %w", filePath, err)
		}

		fileName := filepath.Base(filePath)
		fileHeader := textproto.MIMEHeader{}
		fileHeader.Set("Content-Type", "application/octet-stream")
		fileHeader.Set("Content-Transfer-Encoding", "base64")
		fileHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

		filePart, err := writer.CreatePart(fileHeader)
		if err != nil {
			return nil, err
		}

		encoded := base64.StdEncoding.EncodeToString(fileData)
		// 76자마다 줄바꿈 (RFC 2045)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			filePart.Write([]byte(encoded[i:end] + "\r\n"))
		}
	}

	writer.Close()
	return buf.Bytes(), nil
}

func sendMail(to, subject, body string, files []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	from := cfg.GmailFrom
	password := cfg.GmailPassword

	msg, err := buildMessage(from, to, subject, body, files)
	if err != nil {
		return fmt.Errorf("메시지 생성 실패: %w", err)
	}

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
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	w.Close()
	client.Quit()
	return nil
}
