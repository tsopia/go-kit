package crypto

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignAndParseJWT(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-jwt-signing",
		JWTExpiry: time.Hour,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		claims  JWTClaims
		wantErr bool
	}{
		{
			name: "valid claims with subject",
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user123",
				},
			},
			wantErr: false,
		},
		{
			name: "valid claims with custom fields",
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:  "user456",
					Issuer:   "test-issuer",
					Audience: jwt.ClaimStrings{"test-audience"},
				},
				Custom: map[string]interface{}{
					"role": "admin",
					"org":  "acme",
				},
			},
			wantErr: false,
		},
		{
			name: "valid claims with explicit expiry",
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user789",
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
				},
			},
			wantErr: false,
		},
		{
			name: "valid claims without expiry - should use default",
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user000",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := client.SignJWT(tt.claims)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Parse and verify
			parsed, err := client.ParseJWT(token)
			require.NoError(t, err)
			assert.Equal(t, tt.claims.Subject, parsed.Subject)
			assert.Equal(t, tt.claims.Issuer, parsed.Issuer)
			assert.NotNil(t, parsed.ExpiresAt)

			// Check custom fields
			if tt.claims.Custom != nil {
				for k, v := range tt.claims.Custom {
					assert.Equal(t, v, parsed.Custom[k])
				}
			}
		})
	}
}

func TestSignJWTWithAlg(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-jwt-signing",
		JWTExpiry: time.Hour,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		alg     string
		claims  JWTClaims
		wantErr bool
	}{
		{
			name: "HS256 algorithm",
			alg:  JWTAlgHS256,
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user123",
				},
			},
			wantErr: false,
		},
		{
			name: "HS384 algorithm",
			alg:  JWTAlgHS384,
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user456",
				},
			},
			wantErr: false,
		},
		{
			name: "HS512 algorithm",
			alg:  JWTAlgHS512,
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user789",
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported algorithm",
			alg:  "RS256",
			claims: JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user000",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := client.SignJWTWithAlg(tt.claims, tt.alg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Parse and verify
			parsed, err := client.ParseJWT(token)
			require.NoError(t, err)
			assert.Equal(t, tt.claims.Subject, parsed.Subject)
		})
	}
}

func TestParseJWT_InvalidToken(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-jwt-signing",
		JWTExpiry: time.Hour,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
		errIs   error
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
			errIs:   ErrInvalidToken,
		},
		{
			name:    "malformed token",
			token:   "invalid.token.here",
			wantErr: true,
			errIs:   ErrInvalidToken,
		},
		{
			name:    "token with wrong signature",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.wrongsignature",
			wantErr: true,
			errIs:   ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ParseJWT(tt.token)
			require.Error(t, err)
			if tt.errIs != nil {
				assert.ErrorIs(t, err, tt.errIs)
			}
		})
	}
}

func TestParseJWT_ExpiredToken(t *testing.T) {
	config := &Config{
		JWTSecret: "test-secret-key-for-jwt-signing",
		JWTExpiry: time.Hour,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	// Create an expired token
	expiredClaims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}

	token, err := client.SignJWT(expiredClaims)
	require.NoError(t, err)

	_, err = client.ParseJWT(token)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestSignJWT_NoDefaultClient(t *testing.T) {
	// Reset default client
	clientMu.Lock()
	oldClient := defaultClient
	defaultClient = nil
	clientMu.Unlock()

	defer func() {
		clientMu.Lock()
		defaultClient = oldClient
		clientMu.Unlock()
	}()

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user123",
		},
	}

	_, err := SignJWT(claims)
	assert.ErrorIs(t, err, ErrMissingClient)
}

func TestParseJWT_NoDefaultClient(t *testing.T) {
	// Reset default client
	clientMu.Lock()
	oldClient := defaultClient
	defaultClient = nil
	clientMu.Unlock()

	defer func() {
		clientMu.Lock()
		defaultClient = oldClient
		clientMu.Unlock()
	}()

	_, err := ParseJWT("some.token.here")
	assert.ErrorIs(t, err, ErrMissingClient)
}

func TestSignJWT_NoSecret(t *testing.T) {
	config := &Config{
		JWTSecret: "",
		JWTExpiry: time.Hour,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "user123",
		},
	}

	_, err = client.SignJWT(claims)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestParseJWT_NoSecret(t *testing.T) {
	config := &Config{
		JWTSecret: "",
		JWTExpiry: time.Hour,
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	_, err = client.ParseJWT("some.token.here")
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestSetDefaultExpiry(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		claims       *JWTClaims
		expiry       time.Duration
		expectExpiry bool
	}{
		{
			name: "no expiry set - should set default",
			claims: &JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{},
			},
			expiry:       time.Hour,
			expectExpiry: true,
		},
		{
			name: "expiry already set - should not change",
			claims: &JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Hour)),
				},
			},
			expiry:       time.Hour,
			expectExpiry: true,
		},
		{
			name: "zero expiry - should not set",
			claims: &JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{},
			},
			expiry:       0,
			expectExpiry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDefaultExpiry(tt.claims, tt.expiry)
			if tt.expectExpiry {
				assert.NotNil(t, tt.claims.ExpiresAt)
			} else {
				assert.Nil(t, tt.claims.ExpiresAt)
			}
		})
	}
}
