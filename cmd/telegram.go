package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "텔레그램 메시지/파일 전송",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Println("❌", err)
			os.Exit(1)
		}
		if cfg.TelegramToken == "" || cfg.TelegramChatID == "" {
			fmt.Println("❌ 텔레그램 설정 없음. 'send config set --tg-token TOKEN --tg-chat-id CHAT_ID' 로 설정해줘!")
			os.Exit(1)
		}

		text, _ := cmd.Flags().GetString("text")
		audio, _ := cmd.Flags().GetString("audio")

		if text == "" && len(args) > 0 {
			text = args[0]
		}

		if audio != "" {
			if err := sendTelegramAudio(cfg.TelegramToken, cfg.TelegramChatID, audio, text); err != nil {
				fmt.Println("❌ 오디오 전송 실패:", err)
				os.Exit(1)
			}
			fmt.Println("✅ 텔레그램 오디오 전송 완료!")
			return
		}

		if text == "" {
			fmt.Println("❌ 메시지를 입력해줘! (예: send telegram \"안녕\")")
			os.Exit(1)
		}

		if err := sendTelegram(cfg.TelegramToken, cfg.TelegramChatID, text); err != nil {
			fmt.Println("❌ 전송 실패:", err)
			os.Exit(1)
		}
		fmt.Println("✅ 텔레그램 전송 완료!")
	},
}

func sendTelegram(token, chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func sendTelegramAudio(token, chatID, filePath, caption string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("파일 열기 실패: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	_ = w.WriteField("chat_id", chatID)
	if caption != "" {
		_ = w.WriteField("caption", caption)
	}

	fw, err := w.CreateFormFile("audio", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return err
	}
	w.Close()

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendAudio", token)
	resp, err := http.Post(url, w.FormDataContentType(), &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func init() {
	rootCmd.AddCommand(telegramCmd)
	telegramCmd.Flags().StringP("text", "t", "", "전송할 메시지")
	telegramCmd.Flags().StringP("audio", "a", "", "전송할 오디오 파일 경로 (.mp3)")
}
