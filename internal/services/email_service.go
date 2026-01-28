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

func (s *EmailService) SendEmail(to []string, subject string, body string) error {
	smtpServer := viper.GetString("SMTP_SERVER")
	smtpPort := viper.GetString("SMTP_PORT")
	smtpUser := viper.GetString("SMTP_USER")
	smtpPass := viper.GetString("SMTP_PASS")
	fromEmail := viper.GetString("SMTP_FROM_EMAIL")

	addr := fmt.Sprintf("%s:%s", smtpServer, smtpPort)
	log.Printf("[EmailService] Conectando a %s (SSL)...", addr)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer,
	}

	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		log.Printf("[EmailService] Error crítico al conectar: %v", err)
		return fmt.Errorf("falló tls.DialWithDialer: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(60 * time.Second))

	log.Println("[EmailService] Creando cliente SMTP...")
	client, err := smtp.NewClient(conn, smtpServer)
	if err != nil {
		return fmt.Errorf("falló smtp.NewClient: %w", err)
	}
	defer client.Quit()

	log.Println("[EmailService] Autenticando...")
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpServer)
	if err = client.Auth(auth); err != nil {
		log.Printf("[EmailService] Error en Auth: %v", err)
		return fmt.Errorf("falló autenticación: %w", err)
	}

	log.Println("[EmailService] Configurando remitente y destinatarios...")
	if err = client.Mail(fromEmail); err != nil {
		return fmt.Errorf("falló comando MAIL FROM: %w", err)
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("falló comando RCPT TO (%s): %w", addr, err)
		}
	}

	log.Println("[EmailService] Enviando DATA...")
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("falló comando DATA: %w", err)
	}

	headers := make(map[string]string)
	headers["From"] = fromEmail
	headers["To"] = to[0]
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
		return fmt.Errorf("falló escribiendo contenido: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("falló cerrando data writer: %w", err)
	}

	log.Println("[EmailService] Correo enviado exitosamente.")
	return nil
}
