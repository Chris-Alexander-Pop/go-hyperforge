package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
)

// BuildMIME constructs a simple multipart MIME message suitable for SMTP or SES Raw.
func BuildMIME(msg *Message) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	var header bytes.Buffer
	fmt.Fprintf(&header, "From: %s\r\n", msg.From)
	fmt.Fprintf(&header, "To: %s\r\n", strings.Join(msg.To, ", "))
	if len(msg.CC) > 0 {
		fmt.Fprintf(&header, "Cc: %s\r\n", strings.Join(msg.CC, ", "))
	}
	if msg.ReplyTo != "" {
		fmt.Fprintf(&header, "Reply-To: %s\r\n", msg.ReplyTo)
	}
	fmt.Fprintf(&header, "Subject: %s\r\n", msg.Subject)
	fmt.Fprintf(&header, "MIME-Version: 1.0\r\n")

	if len(msg.Attachments) == 0 && msg.Body.HTML == "" {
		fmt.Fprintf(&header, "Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		header.WriteString(msg.Body.PlainText)
		return header.Bytes(), nil
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if msg.Body.HTML != "" && msg.Body.PlainText != "" {
		var altBuf bytes.Buffer
		alt := multipart.NewWriter(&altBuf)
		if err := writeTextPart(alt, "text/plain; charset=UTF-8", msg.Body.PlainText); err != nil {
			return nil, err
		}
		if err := writeTextPart(alt, "text/html; charset=UTF-8", msg.Body.HTML); err != nil {
			return nil, err
		}
		if err := alt.Close(); err != nil {
			return nil, err
		}
		altHeader := make(textproto.MIMEHeader)
		altHeader.Set("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, alt.Boundary()))
		part, err := writer.CreatePart(altHeader)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(altBuf.Bytes()); err != nil {
			return nil, err
		}
	} else if msg.Body.HTML != "" {
		if err := writeTextPart(writer, "text/html; charset=UTF-8", msg.Body.HTML); err != nil {
			return nil, err
		}
	} else if msg.Body.PlainText != "" {
		if err := writeTextPart(writer, "text/plain; charset=UTF-8", msg.Body.PlainText); err != nil {
			return nil, err
		}
	}

	for _, att := range msg.Attachments {
		h := make(textproto.MIMEHeader)
		ct := att.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		h.Set("Content-Type", ct)
		h.Set("Content-Transfer-Encoding", "base64")
		disposition := "attachment"
		if att.Inline {
			disposition = "inline"
		}
		h.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, att.Filename))
		if att.ContentID != "" {
			h.Set("Content-ID", fmt.Sprintf("<%s>", att.ContentID))
		}
		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, err
		}
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(att.Content)))
		base64.StdEncoding.Encode(encoded, att.Content)
		if _, err := part.Write(encoded); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	fmt.Fprintf(&header, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", writer.Boundary())
	header.Write(body.Bytes())
	return header.Bytes(), nil
}

func writeTextPart(w *multipart.Writer, contentType, body string) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", contentType)
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	qp := quotedprintable.NewWriter(part)
	if _, err := io.WriteString(qp, body); err != nil {
		return err
	}
	return qp.Close()
}
