package gosoap

import (
	"crypto/sha1"
	"encoding/base64"
	"testing"
	"time"

	"go.viam.com/test"
)

// digestFor recomputes the WS-Security password digest from the emitted token
// fields, mirroring how a camera validates it:
// B64ENCODE( SHA1( B64DECODE( Nonce ) + Created + Password ) ).
func digestFor(nonce, created, password string) string {
	decodedNonce, _ := base64.StdEncoding.DecodeString(nonce)
	hasher := sha1.New()
	hasher.Write([]byte(string(decodedNonce) + created + password))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

func TestNewSecurity(t *testing.T) {
	t.Run("zero offset uses local clock", func(t *testing.T) {
		sec := newSecurity("user", "pass", 0)

		created, err := time.Parse(time.RFC3339Nano, sec.Auth.Created)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, time.Since(created).Abs(), test.ShouldBeLessThan, 10*time.Second)

		test.That(t, sec.Auth.Password.Password, test.ShouldEqual,
			digestFor(sec.Auth.Nonce.Nonce, sec.Auth.Created, "pass"))
	})

	t.Run("clock offset shifts Created into device time frame", func(t *testing.T) {
		// A device whose clock reads 2000-01-01 while the local clock is current.
		offset := -26 * 365 * 24 * time.Hour
		sec := newSecurity("user", "pass", offset)

		created, err := time.Parse(time.RFC3339Nano, sec.Auth.Created)
		test.That(t, err, test.ShouldBeNil)
		expected := time.Now().UTC().Add(offset)
		test.That(t, created.Sub(expected).Abs(), test.ShouldBeLessThan, 10*time.Second)

		// The digest must be computed over the shifted timestamp, otherwise the
		// device's digest verification would fail even if it accepted the time.
		test.That(t, sec.Auth.Password.Password, test.ShouldEqual,
			digestFor(sec.Auth.Nonce.Nonce, sec.Auth.Created, "pass"))
	})
}
