package auth

import (
	"encoding/base64"

	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
)

// TOTPSetup carries the data returned to the client to enroll an authenticator.
type TOTPSetup struct {
	Secret     string `json:"-"`           // raw base32 secret, stored encrypted by the caller
	OtpauthURL string `json:"otpauth_url"` // otpauth://... provisioning URI
	QR         string `json:"qr"`          // data:image/png;base64,... QR of the URI
}

// GenerateTOTP creates a new TOTP secret for the given account and issuer, and
// renders a QR data URI for the provisioning URL.
func GenerateTOTP(issuer, account string) (*TOTPSetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
	})
	if err != nil {
		return nil, err
	}
	png, err := qrcode.Encode(key.URL(), qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	return &TOTPSetup{
		Secret:     key.Secret(),
		OtpauthURL: key.URL(),
		QR:         dataURI,
	}, nil
}

// ValidateTOTP checks a 6-digit code against a base32 secret.
func ValidateTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}
