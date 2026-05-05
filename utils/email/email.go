package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type Message struct {
	To      string
	Subject string
	Body    string
}

func Send(config Config, msg Message) error {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	headers := make(map[string]string)
	headers["From"] = config.From
	headers["To"] = msg.To
	headers["Subject"] = msg.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var mailMsg strings.Builder
	for k, v := range headers {
		mailMsg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	mailMsg.WriteString("\r\n")
	mailMsg.WriteString(msg.Body)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: config.Host,
	})
	if err != nil {
		return fmt.Errorf("TLS连接失败: %v", err)
	}

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %v", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP认证失败: %v", err)
	}

	if err = client.Mail(config.From); err != nil {
		return fmt.Errorf("设置发件人失败: %v", err)
	}

	if err = client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("设置收件人失败: %v", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("发送邮件数据失败: %v", err)
	}

	_, err = w.Write([]byte(mailMsg.String()))
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %v", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("关闭邮件写入失败: %v", err)
	}

	return client.Quit()
}

func SendBatch(config Config, toList []string, msg Message) error {
	var errMsgs []string
	for _, to := range toList {
		to = strings.TrimSpace(to)
		if to == "" {
			continue
		}
		msg.To = to
		if err := Send(config, msg); err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", to, err))
		}
	}
	if len(errMsgs) > 0 {
		return fmt.Errorf("部分邮件发送失败: %s", strings.Join(errMsgs, "; "))
	}
	return nil
}
