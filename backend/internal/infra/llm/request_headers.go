package llm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type HeaderTemplateValues struct {
	UserID         uint
	ConversationID uint
}

func modelListHeaderTemplateValues() HeaderTemplateValues {
	return HeaderTemplateValues{UserID: 1, ConversationID: 1}
}

// signJWTAuto replaces "auto" values in X-User-Token with an actual HS256 JWT.
// It returns a copy of raw with "auto" replaced by the signed token, or the original
// raw if no substitution was needed.
func signJWTAuto(raw string, userID uint, secret string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return raw, nil
	}
	parsed := make(map[string]interface{})
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return raw, nil // not JSON – skip
	}
	modified := false
	for key, v := range parsed {
		if strings.EqualFold(strings.TrimSpace(key), "X-User-Token") {
			if s, ok := v.(string); ok && strings.TrimSpace(s) == "auto" {
				if userID == 0 {
					return "", fmt.Errorf("X-User-Token=auto requires a valid user_id")
				}
				if secret == "" {
					return "", fmt.Errorf("X-User-Token=auto requires JWT_SECRET to be configured")
				}
				token, err := signJWT(userID, secret)
				if err != nil {
					return "", fmt.Errorf("signing X-User-Token JWT: %w", err)
				}
				parsed[key] = token
				modified = true
			}
		}
	}
	if !modified {
		return raw, nil
	}
	out, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("re-serializing headers after JWT signing: %w", err)
	}
	return string(out), nil
}

func signJWT(userID uint, secret string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":    strconv.FormatUint(uint64(userID), 10),
		"tenant": "withyou",
		"iss":    "deeix-chat",
		"exp":    now.Add(5 * time.Minute).Unix(),
		"iat":    now.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func setAdditionalHeaders(req *http.Request, raw string, templateValues ...HeaderTemplateValues) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	parsed := make(map[string]interface{})
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return fmt.Errorf("invalid additional headers JSON: %w", err)
	}
	values := HeaderTemplateValues{}
	if len(templateValues) > 0 {
		values = templateValues[0]
	}
	for key, rawValue := range parsed {
		headerKey := strings.TrimSpace(key)
		if headerKey == "" {
			continue
		}
		if hasTemplateSyntax(headerKey) {
			return fmt.Errorf("header name templates are not supported")
		}
		rendered, err := renderHeaderTemplate(stringify(rawValue), values)
		if err != nil {
			return fmt.Errorf("render header %s: %w", headerKey, err)
		}
		req.Header.Set(headerKey, rendered)
	}
	return nil
}

func renderHeaderTemplate(value string, values HeaderTemplateValues) (string, error) {
	if strings.Contains(value, "{{user_id}}") {
		if values.UserID == 0 {
			return "", fmt.Errorf("user_id is unavailable")
		}
		value = strings.ReplaceAll(value, "{{user_id}}", strconv.FormatUint(uint64(values.UserID), 10))
	}
	if strings.Contains(value, "{{conversation_id}}") {
		if values.ConversationID == 0 {
			value = strings.ReplaceAll(value, "{{conversation_id}}", "")
		} else {
			value = strings.ReplaceAll(value, "{{conversation_id}}", strconv.FormatUint(uint64(values.ConversationID), 10))
		}
	}
	if hasTemplateSyntax(value) {
		return "", fmt.Errorf("unsupported header template")
	}
	return value, nil
}

func hasTemplateSyntax(value string) bool {
	return strings.Contains(value, "{{") || strings.Contains(value, "}}")
}
