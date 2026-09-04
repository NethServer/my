/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package email

import (
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func testEmailService() *EmailService {
	return &EmailService{
		host:     "email-smtp.eu-west-1.amazonaws.com",
		port:     587,
		from:     "no-reply@nethesis.it",
		fromName: "My Nethesis",
	}
}

func buildTestMessage(t *testing.T, data EmailData) (string, *mail.Message) {
	t.Helper()

	raw, err := testEmailService().buildMessage(data, time.Date(2026, 9, 4, 10, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}

	message, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("message is not RFC 5322 parsable: %v", err)
	}

	return string(raw), message
}

// crlf mirrors what the quoted-printable encoder does to line endings: the wire
// format is CRLF, whatever the templates use.
func crlf(body string) string {
	return strings.ReplaceAll(body, "\n", "\r\n")
}

func decodeParts(t *testing.T, message *mail.Message) map[string]string {
	t.Helper()

	mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("Content-Type: %v", err)
	}
	if mediaType != "multipart/alternative" {
		t.Fatalf("media type = %q, want multipart/alternative", mediaType)
	}

	decoded := make(map[string]string)
	reader := multipart.NewReader(message.Body, params["boundary"])
	for {
		part, err := reader.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}

		partType, partParams, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("part Content-Type: %v", err)
		}
		if got := strings.ToUpper(partParams["charset"]); got != "UTF-8" {
			t.Errorf("%s charset = %q, want UTF-8", partType, got)
		}
		if got := part.Header.Get("Content-Transfer-Encoding"); got != "quoted-printable" {
			t.Errorf("%s Content-Transfer-Encoding = %q, want quoted-printable", partType, got)
		}

		body, err := io.ReadAll(quotedprintable.NewReader(part))
		if err != nil {
			t.Fatalf("decode %s part: %v", partType, err)
		}
		decoded[partType] = string(body)
	}

	return decoded
}

func TestBuildMessagePreservesUTF8Bodies(t *testing.T) {
	data := EmailData{
		To:       "edoardo.spadoni+mime@nethesis.it",
		Subject:  "Benvenuto su Società Àcme - Account Creato",
		TextBody: "Ciao, èccoti le credenziali. Perché? 🎉\nRuolo: Amministratore",
		HTMLBody: "<p>Ciao, èccoti le credenziali. Perché? 🎉</p>",
	}

	raw, message := buildTestMessage(t, data)

	for i := 0; i < len(raw); i++ {
		if raw[i] > 127 && i < strings.Index(raw, "\r\n\r\n") {
			t.Fatalf("header block contains raw non-ASCII byte at offset %d", i)
		}
	}

	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("decode Subject: %v", err)
	}
	if subject != data.Subject {
		t.Errorf("Subject = %q, want %q", subject, data.Subject)
	}

	parts := decodeParts(t, message)
	if got, want := parts["text/plain"], crlf(data.TextBody); got != want {
		t.Errorf("text part = %q, want %q", got, want)
	}
	if got, want := parts["text/html"], crlf(data.HTMLBody); got != want {
		t.Errorf("html part = %q, want %q", got, want)
	}
}

func TestBuildMessageSetsDateFromAndMessageID(t *testing.T) {
	_, message := buildTestMessage(t, EmailData{
		To:       "edoardo.spadoni+mime@nethesis.it",
		Subject:  "Welcome to Acme - Account Created",
		TextBody: "plain",
	})

	date, err := message.Header.Date()
	if err != nil {
		t.Fatalf("Date: %v", err)
	}
	if !date.Equal(time.Date(2026, 9, 4, 10, 30, 0, 0, time.UTC)) {
		t.Errorf("Date = %v, want 2026-09-04 10:30 UTC", date)
	}

	messageID := message.Header.Get("Message-ID")
	if !strings.HasPrefix(messageID, "<") || !strings.HasSuffix(messageID, "@nethesis.it>") {
		t.Errorf("Message-ID = %q, want <token@nethesis.it>", messageID)
	}

	from, err := message.Header.AddressList("From")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if len(from) != 1 || from[0].Address != "no-reply@nethesis.it" || from[0].Name != "My Nethesis" {
		t.Errorf("From = %v, want My Nethesis <no-reply@nethesis.it>", from)
	}
}

func TestBuildMessageNeutralizesSubjectHeaderInjection(t *testing.T) {
	raw, message := buildTestMessage(t, EmailData{
		To:       "edoardo.spadoni+mime@nethesis.it",
		Subject:  "Benvenuto su Acme\r\nBcc: attacker@example.com",
		TextBody: "plain",
	})

	for _, line := range strings.Split(raw, "\r\n") {
		if strings.HasPrefix(line, "Bcc:") {
			t.Errorf("injected Bcc header survived encoding:\n%s", raw)
		}
	}
	if !strings.Contains(message.Header.Get("Subject"), "=0D=0A") {
		t.Errorf("Subject = %q, want the CRLF encoded", message.Header.Get("Subject"))
	}
	if message.Header.Get("Bcc") != "" {
		t.Error("message exposes a Bcc header")
	}
}

func TestBuildMessageOmitsEmptyParts(t *testing.T) {
	_, message := buildTestMessage(t, EmailData{
		To:       "edoardo.spadoni+mime@nethesis.it",
		Subject:  "Welcome to Acme - Account Created",
		TextBody: "plain only",
	})

	parts := decodeParts(t, message)
	if len(parts) != 1 {
		t.Errorf("parts = %d, want 1", len(parts))
	}
	if _, ok := parts["text/html"]; ok {
		t.Error("empty HTML body produced a part")
	}
}
