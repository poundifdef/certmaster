package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"

	"github.com/poundifdef/certmaster/models"

	"github.com/go-acme/lego/v4/certificate"
)

type Destination struct {
	To           string `mapstructure:"to" description:"Email address of where to send certificates"`
	From         string `credential:"true" mapstructure:"from" description:"Email of from address"`
	SMTPHost     string `credential:"true" mapstructure:"host" description:"SMTP Host"`
	SMTPPort     string `credential:"true" mapstructure:"port" description:"SMTP Port"`
	SMTPUsername string `credential:"true" mapstructure:"username" description:"SMTP Username"`
	SMTPPassword string `credential:"true" mapstructure:"password" description:"SMTP Password"`
}

func (d Destination) Upload(request models.CertRequest, cert *certificate.Resource) error {
	subject := "Certificate for " + request.Domain
	body := fmt.Sprintf(`
	Attached is your certificate for %s
	`, request.Domain)

	auth := smtp.PlainAuth("", d.SMTPUsername, d.SMTPPassword, d.SMTPHost)

	// Create a new buffer and a multipart writer for the email body
	var email bytes.Buffer
	writer := multipart.NewWriter(&email)

	// Write the main headers (only once)
	fmt.Fprintf(&email, "From: %s\r\n", d.From)
	fmt.Fprintf(&email, "To: %s\r\n", d.To)
	fmt.Fprintf(&email, "Subject: %s\r\n", subject)
	fmt.Fprintf(&email, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&email, "Content-Type: multipart/mixed; boundary=%s\r\n", writer.Boundary())
	fmt.Fprintf(&email, "\r\n")

	// Create a part for the email body
	part, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=UTF-8"}})
	if err != nil {
		return err
	}
	_, err = part.Write([]byte(body))
	if err != nil {
		return err
	}

	// Function to add attachments
	addAttachment := func(filename, content string) error {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		h.Set("Content-Type", "text/plain; charset=UTF-8")
		h.Set("Content-Transfer-Encoding", "base64")

		part, err := writer.CreatePart(h)
		if err != nil {
			return err
		}
		encoder := base64.NewEncoder(base64.StdEncoding, part)
		_, err = encoder.Write([]byte(content))
		if err != nil {
			return err
		}
		return encoder.Close()
	}

	// Add the first attachment
	err = addAttachment("cert.txt", string(cert.Certificate))
	if err != nil {
		return err
	}

	// Add the second attachment
	err = addAttachment("private.txt", string(cert.PrivateKey))
	if err != nil {
		return err
	}

	// Close the multipart writer to set the terminating boundary
	if err := writer.Close(); err != nil {
		return err
	}

	// Send the email
	err = smtp.SendMail(d.SMTPHost+":"+d.SMTPPort, auth, d.From, []string{d.To}, email.Bytes())
	if err != nil {
		return err
	}

	return nil
}
