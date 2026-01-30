package services

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type EmailService struct{}

func NewEmailService() *EmailService {
	return &EmailService{}
}

func (s *EmailService) SendEmail(to []string, cc []string, bcc []string, subject string, body string) error {
	smtpServer := viper.GetString("SMTP_SERVER")
	smtpPort := viper.GetString("SMTP_PORT")
	smtpUser := viper.GetString("SMTP_USER")
	smtpPass := viper.GetString("SMTP_PASS")
	fromEmail := viper.GetString("SMTP_FROM_EMAIL")

	addr := fmt.Sprintf("%s:%s", smtpServer, smtpPort)
	log.Printf("[EmailService] Config: %s, User: %s", addr, smtpUser)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer,
	}

	var client *smtp.Client
	var err error

	if smtpPort == "465" {
		log.Printf("[EmailService] Conectando vía SMTPS (Implicit SSL) a %s...", addr)
		dialer := &net.Dialer{Timeout: 30 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("[EmailService] Error SMTPS: %v", err)
			return fmt.Errorf("falló SMTPS dial: %w", err)
		}

		client, err = smtp.NewClient(conn, smtpServer)
		if err != nil {
			conn.Close()
			return fmt.Errorf("falló smtp.NewClient SMTPS: %w", err)
		}
	} else {
		log.Printf("[EmailService] Conectando vía SMTP (Plain/STARTTLS) a %s...", addr)
		conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
		if err != nil {
			log.Printf("[EmailService] Error TCP: %v", err)
			return fmt.Errorf("falló TCP dial: %w", err)
		}

		conn.SetDeadline(time.Now().Add(60 * time.Second))

		client, err = smtp.NewClient(conn, smtpServer)
		if err != nil {
			conn.Close()
			return fmt.Errorf("falló smtp.NewClient SMTP: %w", err)
		}

		if ok, _ := client.Extension("STARTTLS"); ok {
			log.Println("[EmailService] Iniciando STARTTLS...")
			if err = client.StartTLS(tlsConfig); err != nil {
				client.Quit()
				return fmt.Errorf("falló STARTTLS: %w", err)
			}
		}
	}
	defer client.Quit()

	log.Println("[EmailService] Autenticando...")
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpServer)
	if err = client.Auth(auth); err != nil {
		log.Printf("[EmailService] Error Auth: %v", err)
		return fmt.Errorf("autenticación fallida: %w", err)
	}

	if err = client.Mail(fromEmail); err != nil {
		return fmt.Errorf("MAIL FROM error: %w", err)
	}

	recipients := make([]string, 0)
	recipients = append(recipients, to...)
	recipients = append(recipients, cc...)
	recipients = append(recipients, bcc...)
	// Always BCC the sender for records
	recipients = append(recipients, fromEmail)

	for _, addr := range recipients {
		if addr == "" {
			continue
		}
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("RCPT TO error for %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA error: %w", err)
	}

	headers := make(map[string]string)
	headers["From"] = fromEmail
	headers["To"] = strings.Join(to, ", ")
	if len(cc) > 0 {
		headers["Cc"] = strings.Join(cc, ", ")
	}
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	var message strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&message, "%s: %s\r\n", k, v)
	}
	message.WriteString("\r\n" + body)

	_, err = w.Write([]byte(message.String()))
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed closing data: %w", err)
	}

	log.Println("[EmailService] Envío exitoso.")
	return nil
}
