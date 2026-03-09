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
	"strings"

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
		photo, _ := cmd.Flags().GetString("photo")
		video, _ := cmd.Flags().GetString("video")
		file, _ := cmd.Flags().GetString("file")

		if text == "" && len(args) > 0 {
			text = args[0]
		}

		if audio != "" {
			if err := sendTelegramFile(cfg.TelegramToken, cfg.TelegramChatID, "sendAudio", "audio", audio, text); err != nil {
				fmt.Println("❌ 오디오 전송 실패:", err)
				os.Exit(1)
			}
			fmt.Println("✅ 텔레그램 오디오 전송 완료!")
			return
		}

		if photo != "" {
			if err := sendTelegramFile(cfg.TelegramToken, cfg.TelegramChatID, "sendPhoto", "photo", photo, text); err != nil {
				fmt.Println("❌ 이미지 전송 실패:", err)
				os.Exit(1)
			}
			fmt.Println("✅ 텔레그램 이미지 전송 완료!")
			return
		}

		if video != "" {
			if err := sendTelegramFile(cfg.TelegramToken, cfg.TelegramChatID, "sendVideo", "video", video, text); err != nil {
				fmt.Println("❌ 영상 전송 실패:", err)
				os.Exit(1)
			}
			fmt.Println("✅ 텔레그램 영상 전송 완료!")
			return
		}

		if file != "" {
			if err := sendTelegramFile(cfg.TelegramToken, cfg.TelegramChatID, "sendDocument", "document", file, text); err != nil {
				fmt.Println("❌ 파일 전송 실패:", err)
				os.Exit(1)
			}
			fmt.Println("✅ 텔레그램 파일 전송 완료!")
			return
		}

		if text == "" {
			fmt.Println("❌ 메시지를 입력해줘! (예: send telegram \"안녕\")")
			fmt.Println("📎 파일 전송: --audio / --photo / --video / --file")
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

// sendTelegramFile: audio/photo/video/document 공통 전송 함수
// 파일은 경로만 받아서 스트리밍으로 전송 — 바이너리 데이터는 메모리에 올리지 않음
func sendTelegramFile(token, chatID, method, fieldName, filePath, caption string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("파일 열기 실패: %w", err)
	}
	defer f.Close()

	pr, pw := io.Pipe()
	w := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer w.Close()

		_ = w.WriteField("chat_id", chatID)
		if caption != "" {
			_ = w.WriteField("caption", caption)
		}

		fw, err := w.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err = io.Copy(fw, f); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)
	resp, err := http.Post(apiURL, w.FormDataContentType(), pr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		// 에러 메시지만 읽고 바이너리는 버림
		msg := string(b)
		if idx := strings.Index(msg, "\"description\""); idx != -1 {
			msg = msg[idx:]
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(telegramCmd)
	telegramCmd.Flags().StringP("text", "t", "", "전송할 메시지")
	telegramCmd.Flags().StringP("audio", "a", "", "오디오 파일 경로 (.mp3)")
	telegramCmd.Flags().StringP("photo", "p", "", "이미지 파일 경로 (.jpg, .png)")
	telegramCmd.Flags().StringP("video", "v", "", "영상 파일 경로 (.mp4)")
	telegramCmd.Flags().StringP("file", "f", "", "파일 경로 (모든 형식)")
}
