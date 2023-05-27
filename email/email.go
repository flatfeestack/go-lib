package email

import (
	"bytes"
	"fmt"
	"github.com/go-jose/go-jose/v3/json"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"strings"
	"time"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})
}

type SendEmailRequest struct {
	sendgridRequest SendgridRequest
	url             string
	emailFromName   string
	emailFrom       string
	emailToken      string
}

type SendgridRequest struct {
	MailTo      string `json:"mail_to,omitempty"`
	Subject     string `json:"subject"`
	TextMessage string `json:"text_message"`
	HtmlMessage string `json:"html_message"`
}

func SendEmail(sendEmailRequest SendEmailRequest) error {
	c := &http.Client{
		Timeout: 15 * time.Second,
	}

	var jsonData []byte
	var err error
	if strings.Contains(sendEmailRequest.url, "sendgrid") {
		sendGridReq := mail.NewSingleEmail(
			mail.NewEmail(sendEmailRequest.emailFromName, sendEmailRequest.emailFrom),
			sendEmailRequest.sendgridRequest.Subject,
			mail.NewEmail("", sendEmailRequest.sendgridRequest.MailTo),
			sendEmailRequest.sendgridRequest.TextMessage,
			sendEmailRequest.sendgridRequest.HtmlMessage)
		jsonData, err = json.Marshal(sendGridReq)
	} else {
		jsonData, err = json.Marshal(sendEmailRequest.sendgridRequest)
	}

	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", sendEmailRequest.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+sendEmailRequest.emailToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("could not send email: %v %v", resp.Status, resp.StatusCode)
	}
	return nil
}

func PrepareEmail(mailTo string, data map[string]interface{}, templateKey string, defaultSubject string,
	defaultText string, lang string) SendgridRequest {
	textMessage := parseTemplate("plain/"+lang+"/"+templateKey+".txt", data)
	if textMessage == "" {
		textMessage = defaultText
	}

	headerTemplate := parseTemplate("html/"+lang+"/header.html", data)
	footerTemplate := parseTemplate("html/"+lang+"/footer.html", data)
	htmlBody := parseTemplate("html/"+lang+"/"+templateKey+".html", data)
	htmlMessage := headerTemplate + htmlBody + footerTemplate

	return SendgridRequest{
		MailTo:      mailTo,
		Subject:     defaultSubject,
		TextMessage: textMessage,
		HtmlMessage: htmlMessage,
	}
}

func parseTemplate(filename string, other map[string]interface{}) string {
	textMessage := ""
	tmplPlain, err := template.ParseFiles("mail-templates/" + filename)
	if err == nil {
		var buf bytes.Buffer
		err = tmplPlain.Execute(&buf, other)
		if err == nil {
			textMessage = buf.String()
		} else {
			log.Printf("cannot execute template file [%v], err: %v", filename, err)
		}
	} else {
		log.Printf("cannot prepare file template file [%v], err: %v", filename, err)
	}
	return textMessage
}
