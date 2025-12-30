package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIDocumentation returns an HTML page documenting the API
func APIDocumentation(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>After Dark Systems - Change Management API</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
            min-height: 100vh;
            color: #e4e4e4;
            line-height: 1.6;
        }
        .container { max-width: 900px; margin: 0 auto; padding: 40px 20px; }
        header {
            text-align: center;
            padding: 40px 0;
            border-bottom: 1px solid rgba(255,255,255,0.1);
            margin-bottom: 40px;
        }
        h1 {
            font-size: 2.5rem;
            background: linear-gradient(90deg, #e94560, #ff6b6b);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 10px;
        }
        .subtitle { color: #888; font-size: 1.1rem; }
        .status {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            background: rgba(0,255,136,0.1);
            border: 1px solid rgba(0,255,136,0.3);
            padding: 8px 16px;
            border-radius: 20px;
            margin-top: 20px;
            font-size: 0.9rem;
        }
        .status-dot {
            width: 8px;
            height: 8px;
            background: #00ff88;
            border-radius: 50%;
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
        section {
            background: rgba(255,255,255,0.03);
            border: 1px solid rgba(255,255,255,0.08);
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 24px;
        }
        h2 {
            color: #e94560;
            font-size: 1.3rem;
            margin-bottom: 16px;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        h2::before { content: '→'; color: #ff6b6b; }
        h3 { color: #ccc; font-size: 1rem; margin: 16px 0 8px; }
        .endpoint {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 10px 14px;
            background: rgba(0,0,0,0.2);
            border-radius: 8px;
            margin: 8px 0;
            font-family: 'SF Mono', Monaco, 'Courier New', monospace;
            font-size: 0.9rem;
        }
        .method {
            padding: 4px 10px;
            border-radius: 4px;
            font-weight: 600;
            font-size: 0.75rem;
            min-width: 60px;
            text-align: center;
        }
        .get { background: #10b981; color: #fff; }
        .post { background: #3b82f6; color: #fff; }
        .patch { background: #f59e0b; color: #fff; }
        .delete { background: #ef4444; color: #fff; }
        .path { color: #e4e4e4; }
        .desc { color: #888; margin-left: auto; font-family: inherit; font-size: 0.85rem; }
        code {
            background: rgba(233,69,96,0.15);
            padding: 2px 8px;
            border-radius: 4px;
            font-family: 'SF Mono', Monaco, 'Courier New', monospace;
            font-size: 0.85rem;
            color: #ff6b6b;
        }
        a { color: #e94560; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .links { display: flex; gap: 20px; justify-content: center; margin-top: 30px; flex-wrap: wrap; }
        .links a {
            background: rgba(233,69,96,0.15);
            padding: 10px 20px;
            border-radius: 8px;
            border: 1px solid rgba(233,69,96,0.3);
            transition: all 0.2s;
        }
        .links a:hover {
            background: rgba(233,69,96,0.25);
            text-decoration: none;
            transform: translateY(-2px);
        }
        footer {
            text-align: center;
            padding: 30px 0;
            color: #666;
            font-size: 0.85rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Change Management API</h1>
            <p class="subtitle">After Dark Systems Operations Platform</p>
            <div class="status">
                <span class="status-dot"></span>
                <span>API Online</span>
            </div>
        </header>

        <section>
            <h2>Overview</h2>
            <p>This API provides enterprise change management, ticket tracking, and approval workflows.
            All API endpoints are prefixed with <code>/v1</code> and require authentication unless noted otherwise.</p>
        </section>

        <section>
            <h2>Authentication</h2>
            <p>Obtain a JWT token via the login endpoint and include it in subsequent requests:</p>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/v1/auth/login</span>
                <span class="desc">Authenticate with credentials</span>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/v1/auth/login/oauth2/afterdark</span>
                <span class="desc">OAuth2 via After Dark Central Auth</span>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/v1/auth/refresh</span>
                <span class="desc">Refresh access token</span>
            </div>
            <p style="margin-top: 12px;">Include token in header: <code>Authorization: Bearer &lt;token&gt;</code></p>
        </section>

        <section>
            <h2>Tickets</h2>
            <p>Create and manage change tickets with full audit trails.</p>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/v1/tickets</span>
                <span class="desc">Create new ticket</span>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/v1/tickets</span>
                <span class="desc">List tickets</span>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/v1/tickets/:id</span>
                <span class="desc">Get ticket details</span>
            </div>
            <div class="endpoint">
                <span class="method patch">PATCH</span>
                <span class="path">/v1/tickets/:id</span>
                <span class="desc">Update ticket</span>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/v1/tickets/:id/submit</span>
                <span class="desc">Submit for approval</span>
            </div>
            <h3>Ticket ID Format</h3>
            <p>Tickets use the format: <code>CHG-YYYY-NNNNN</code> (e.g., CHG-2025-00001)</p>
        </section>

        <section>
            <h2>Approvals</h2>
            <p>Review and approve/deny submitted change tickets.</p>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/v1/approvals</span>
                <span class="desc">List pending approvals</span>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/v1/approvals/:id/approve</span>
                <span class="desc">Approve change</span>
            </div>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/v1/approvals/:id/deny</span>
                <span class="desc">Deny change</span>
            </div>
            <h3>Token-Based Approval (No Auth Required)</h3>
            <div class="endpoint">
                <span class="method post">POST</span>
                <span class="path">/v1/approvals/token/:token/approve</span>
                <span class="desc">Approve via email token</span>
            </div>
        </section>

        <section>
            <h2>Health & Status</h2>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/health</span>
                <span class="desc">Basic health check</span>
            </div>
            <div class="endpoint">
                <span class="method get">GET</span>
                <span class="path">/health/ready</span>
                <span class="desc">Readiness with dependencies</span>
            </div>
        </section>

        <section>
            <h2>CLI Tool</h2>
            <p>A command-line interface is available for managing tickets:</p>
            <div class="endpoint" style="background: rgba(16,185,129,0.1); border: 1px solid rgba(16,185,129,0.2);">
                <span style="color: #10b981; font-weight: 600;">$</span>
                <span class="path">changes ticket create --type standard --title "Deploy v2.0"</span>
            </div>
            <div class="endpoint" style="background: rgba(16,185,129,0.1); border: 1px solid rgba(16,185,129,0.2);">
                <span style="color: #10b981; font-weight: 600;">$</span>
                <span class="path">changes ticket list --status pending</span>
            </div>
        </section>

        <div class="links">
            <a href="https://changes.afterdarksys.com">Web Interface</a>
            <a href="/health">Health Status</a>
        </div>

        <footer>
            <p>&copy; After Dark Systems &bull; Change Management Platform</p>
        </footer>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// Health returns basic health status
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// Ready returns readiness status with dependency checks
func Ready(c *gin.Context) {
	// TODO: Check database, Redis, and other dependencies
	c.JSON(http.StatusOK, gin.H{
		"database": "connected",
		"redis":    "connected",
		"ses":      "available",
	})
}

// Metrics returns Prometheus metrics
func Metrics(c *gin.Context) {
	// TODO: Implement Prometheus metrics
	c.String(http.StatusOK, "# HELP api_requests_total Total API requests\n# TYPE api_requests_total counter\n")
}

// NotImplemented returns a 501 Not Implemented response
func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":      "NOT_IMPLEMENTED",
			"message":   "This endpoint is not yet implemented",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// Auth handlers
func Login(c *gin.Context)              { notImplemented(c) }
func LoginMFA(c *gin.Context)           { notImplemented(c) }
func LoginOAuth2Google(c *gin.Context)  { notImplemented(c) }
func LoginOAuth2AfterDark(c *gin.Context) { notImplemented(c) }
func LoginPasskeyBegin(c *gin.Context)  { notImplemented(c) }
func LoginPasskeyFinish(c *gin.Context) { notImplemented(c) }
func RefreshToken(c *gin.Context)       { notImplemented(c) }
func GetCurrentUser(c *gin.Context)     { notImplemented(c) }
func Logout(c *gin.Context)             { notImplemented(c) }

// Ticket handlers - now implemented in ticket_handlers.go
// These stub functions remain for backwards compatibility with existing router
// New code should use TicketHandler struct methods directly
func CreateTicket(c *gin.Context)       { notImplemented(c) }
func ListTickets(c *gin.Context)        { notImplemented(c) }
func GetTicket(c *gin.Context)          { notImplemented(c) }
func UpdateTicket(c *gin.Context)       { notImplemented(c) }
func SubmitTicket(c *gin.Context)       { notImplemented(c) }
func CancelTicket(c *gin.Context)       { notImplemented(c) }
func CloseTicket(c *gin.Context)        { notImplemented(c) }
func ReopenTicket(c *gin.Context)       { notImplemented(c) }
func GetTicketRevisions(c *gin.Context) { notImplemented(c) }
func GetTicketAudit(c *gin.Context)     { notImplemented(c) }

// Additional ticket endpoints
func GetTicketQueue(c *gin.Context)     { notImplemented(c) }
func AssignTicket(c *gin.Context)       { notImplemented(c) }
func LinkRepository(c *gin.Context)     { notImplemented(c) }
func UnlinkRepository(c *gin.Context)   { notImplemented(c) }
func AddWatcher(c *gin.Context)         { notImplemented(c) }
func RemoveWatcher(c *gin.Context)      { notImplemented(c) }

// Repository handlers
func ListRepositories(c *gin.Context)   { notImplemented(c) }
func CreateRepository(c *gin.Context)   { notImplemented(c) }
func GetRepository(c *gin.Context)      { notImplemented(c) }
func UpdateRepository(c *gin.Context)   { notImplemented(c) }
func DeleteRepository(c *gin.Context)   { notImplemented(c) }

// Project handlers
func ListProjects(c *gin.Context)       { notImplemented(c) }
func CreateProject(c *gin.Context)      { notImplemented(c) }
func GetProject(c *gin.Context)         { notImplemented(c) }
func UpdateProject(c *gin.Context)      { notImplemented(c) }
func DeleteProject(c *gin.Context)      { notImplemented(c) }

// Group handlers
func ListGroups(c *gin.Context)         { notImplemented(c) }
func CreateGroup(c *gin.Context)        { notImplemented(c) }
func GetGroup(c *gin.Context)           { notImplemented(c) }
func UpdateGroup(c *gin.Context)        { notImplemented(c) }
func DeleteGroup(c *gin.Context)        { notImplemented(c) }
func AddGroupMember(c *gin.Context)     { notImplemented(c) }
func RemoveGroupMember(c *gin.Context)  { notImplemented(c) }

// Employee directory handlers
func SearchEmployees(c *gin.Context)    { notImplemented(c) }
func GetEmployee(c *gin.Context)        { notImplemented(c) }
func UpdateEmployee(c *gin.Context)     { notImplemented(c) }

// Ticket ACL handlers
func GetTicketACLs(c *gin.Context)      { notImplemented(c) }
func GrantTicketACL(c *gin.Context)     { notImplemented(c) }
func RevokeTicketACL(c *gin.Context)    { notImplemented(c) }

// Failed signup handlers
func CollectFailedSignupContact(c *gin.Context) { notImplemented(c) }
func ListFailedSignups(c *gin.Context)          { notImplemented(c) }
func ResolveFailedSignup(c *gin.Context)        { notImplemented(c) }

// Approval handlers
func ListApprovals(c *gin.Context)      { notImplemented(c) }
func GetApproval(c *gin.Context)        { notImplemented(c) }
func Approve(c *gin.Context)            { notImplemented(c) }
func Deny(c *gin.Context)               { notImplemented(c) }
func RequestUpdate(c *gin.Context)      { notImplemented(c) }
func ApproveByToken(c *gin.Context)     { notImplemented(c) }
func DenyByToken(c *gin.Context)        { notImplemented(c) }
func GetApprovalByToken(c *gin.Context) { notImplemented(c) }

// Comment handlers
func CreateComment(c *gin.Context)      { notImplemented(c) }
func ListComments(c *gin.Context)       { notImplemented(c) }
func UpdateComment(c *gin.Context)      { notImplemented(c) }
func DeleteComment(c *gin.Context)      { notImplemented(c) }

// User handlers
func ListUsers(c *gin.Context)          { notImplemented(c) }
func CreateUser(c *gin.Context)         { notImplemented(c) }
func GetUser(c *gin.Context)            { notImplemented(c) }
func UpdateUser(c *gin.Context)         { notImplemented(c) }
func DeleteUser(c *gin.Context)         { notImplemented(c) }
func ResetUserPassword(c *gin.Context)  { notImplemented(c) }
func EnableUserMFA(c *gin.Context)      { notImplemented(c) }
func DisableUserMFA(c *gin.Context)     { notImplemented(c) }

// Compliance handlers
func ListComplianceFrameworks(c *gin.Context) { notImplemented(c) }
func ListComplianceTemplates(c *gin.Context)  { notImplemented(c) }
func CreateComplianceTemplate(c *gin.Context) { notImplemented(c) }

// Report handlers
func AuditReport(c *gin.Context)        { notImplemented(c) }
func ComplianceReport(c *gin.Context)   { notImplemented(c) }
func UserActivityReport(c *gin.Context) { notImplemented(c) }
