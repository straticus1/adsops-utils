module github.com/afterdarksys/adsops-utils

go 1.21

require (
	// Web framework
	github.com/gin-gonic/gin v1.10.0

	// Database
	github.com/jackc/pgx/v5 v5.5.0
	github.com/jmoiron/sqlx v1.3.5

	// Redis
	github.com/redis/go-redis/v9 v9.3.0

	// Authentication
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/coreos/go-oidc/v3 v3.9.0
	github.com/go-webauthn/webauthn v0.10.1
	golang.org/x/oauth2 v0.15.0

	// Validation
	github.com/go-playground/validator/v10 v10.16.0

	// CLI
	github.com/spf13/cobra v1.8.0
	github.com/charmbracelet/bubbletea v0.25.0
	github.com/charmbracelet/lipgloss v0.9.1

	// AWS SDK
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/config v1.26.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.0
	github.com/aws/aws-sdk-go-v2/service/ses v1.21.0
	github.com/aws/aws-sdk-go-v2/service/sqs v1.29.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.26.0

	// Logging
	go.uber.org/zap v1.26.0

	// Configuration
	github.com/spf13/viper v1.18.2

	// Email
	github.com/matcornic/hermes/v2 v2.1.0

	// Cryptography
	golang.org/x/crypto v0.17.0

	// Utilities
	github.com/google/uuid v1.5.0
	github.com/pkg/errors v0.9.1
)
