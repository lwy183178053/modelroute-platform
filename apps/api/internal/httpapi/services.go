package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

const (
	roleCommon = 1
	roleAdmin  = 10
)

type SessionManager interface {
	Create(context.Context, platformauth.Session) (string, error)
	Get(context.Context, string) (platformauth.Session, error)
	Save(context.Context, string, platformauth.Session) error
	Delete(context.Context, string) error
	DeleteUserSessions(context.Context, int) error
}

type NewAPIClient interface {
	Authenticate(context.Context, string, any, string) (newapi.AuthenticationResult, error)
	Do(context.Context, string, string, any, *newapi.Auth) (json.RawMessage, error)
	Proxy(context.Context, string, string, http.Header, []byte) (*http.Response, error)
}

type AuditRecorder interface {
	Record(context.Context, AuditEntry) error
	List(context.Context, int, int) ([]AuditEntry, int, error)
}

type UsageAliasResolver interface {
	PublicModelNames(context.Context, []string) (map[string]string, error)
}

type AdminOverviewReader interface {
	Snapshot(context.Context, time.Time) (AdminOverviewSnapshot, error)
}

type AuditEntry struct {
	ID          int64     `json:"id"`
	ActorUserID int       `json:"actor_user_id"`
	ActorRole   string    `json:"actor_role"`
	Action      string    `json:"action"`
	ObjectType  string    `json:"object_type"`
	ObjectID    string    `json:"object_id,omitempty"`
	Result      string    `json:"result"`
	Reason      string    `json:"reason,omitempty"`
	RequestID   string    `json:"request_id,omitempty"`
	SourceIP    string    `json:"source_ip,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CookieConfig struct {
	Name         string
	Secure       bool
	PublicOrigin string
	MaxAge       time.Duration
}

type Services struct {
	Sessions       SessionManager
	NewAPI         NewAPIClient
	AdminNewAPI    AdminNewAPI
	AdminUsername  string
	Audit          AuditRecorder
	Merchants      MerchantRepository
	Credentials    CredentialProtector
	Routing        RoutingRepository
	OfferSync      OfferSynchronizer
	Health         HealthCycleRunner
	Payments       PaymentRepository
	PaymentCredit  HealthCycleRunner
	Settlement     SettlementRepository
	SettlementSync HealthCycleRunner
	UsageAliases   UsageAliasResolver
	Overview       AdminOverviewReader
	Announcements  AnnouncementRepository
	Cookie         CookieConfig
}

type HealthCycleRunner interface {
	RunCycle(context.Context, time.Time) error
}
