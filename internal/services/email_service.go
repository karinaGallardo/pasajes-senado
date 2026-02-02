package services

import (
	"crypto/tls"
	"log"
	"strconv"

	"github.com/go-mail/mail/v2"
	"github.com/spf13/viper"
)

type EmailService struct{}

func NewEmailService() *EmailService {
	return &EmailService{}
}

func (s *EmailService) SendEmail(to []string, cc []string, bcc []string, subject string, body string) error {
	smtpServer := viper.GetString("SMTP_SERVER")
	smtpPortStr := viper.GetString("SMTP_PORT")
	smtpUser := viper.GetString("SMTP_USER")
	smtpPass := viper.GetString("SMTP_PASS")
	fromEmail := viper.GetString("SMTP_FROM_EMAIL")
	smtpUseSSL := viper.GetBool("SMTP_USE_SSL")

	port, _ := strconv.Atoi(smtpPortStr)

	m := mail.NewMessage()
	m.SetHeader("From", fromEmail)
	m.SetHeader("To", to...)
	if len(cc) > 0 {
		m.SetHeader("Cc", cc...)
	}

	fullBcc := append(bcc, fromEmail)
	if len(fullBcc) > 0 {
		m.SetHeader("Bcc", fullBcc...)
	}

	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := mail.NewDialer(smtpServer, port, smtpUser, smtpPass)

	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if port == 465 || smtpUseSSL {
		d.SSL = true
	}

	if smtpUser == "" {
		d.Auth = nil
	}

	log.Printf("[EmailService] Enviando a: %v | Server: %s:%d | SSL: %v", to, smtpServer, port, d.SSL)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("[EmailService] Error enviando correo: %v", err)
		return err
	}

	log.Println("[EmailService] Envío exitoso.")
	return nil
}
