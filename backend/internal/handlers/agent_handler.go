package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"clawreef/internal/services"
	"clawreef/internal/utils"

	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	agentService          services.InstanceAgentService
	commandService        services.InstanceCommandService
	runtimeStatusService  services.InstanceRuntimeStatusService
	configRevisionService services.InstanceConfigRevisionService
}

func NewAgentHandler(agentService services.InstanceAgentService, commandService services.InstanceCommandService, runtimeStatusService services.InstanceRuntimeStatusService, configRevisionService services.InstanceConfigRevisionService) *AgentHandler {
	return &AgentHandler{
		agentService:          agentService,
		commandService:        commandService,
		runtimeStatusService:  runtimeStatusService,
		configRevisionService: configRevisionService,
	}
}

func (h *AgentHandler) Register(c *gin.Context) {
	bootstrapToken := extractBearerToken(c.GetHeader("Authorization"))
	if bootstrapToken == "" {
		utils.Error(c, http.StatusUnauthorized, "Agent bootstrap token is required")
		return
	}

	var req services.AgentRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	resp, err := h.agentService.Register(bootstrapToken, req, c.ClientIP())
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Agent registered successfully", resp)
}

func (h *AgentHandler) Heartbeat(c *gin.Context) {
	session, ok := h.authenticateAgentSession(c)
	if !ok {
		return
	}

	var req services.AgentHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now().UTC()
	}

	resp, err := h.agentService.Heartbeat(session, req, c.ClientIP())
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Agent heartbeat accepted", resp)
}

func (h *AgentHandler) NextCommand(c *gin.Context) {
	session, ok := h.authenticateAgentSession(c)
	if !ok {
		return
	}

	command, err := h.commandService.GetNextForAgent(session)
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Agent next command retrieved successfully", gin.H{"command": command})
}

func (h *AgentHandler) StartCommand(c *gin.Context) {
	session, ok := h.authenticateAgentSession(c)
	if !ok {
		return
	}
	commandID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid command ID")
		return
	}
	var req struct {
		AgentID   string     `json:"agent_id" binding:"required"`
		StartedAt *time.Time `json:"started_at,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if req.AgentID != session.Agent.AgentID {
		utils.Error(c, http.StatusForbidden, "Agent ID does not match session")
		return
	}
	if err := h.commandService.MarkStarted(session, commandID, req.StartedAt); err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Agent command marked as started", nil)
}

func (h *AgentHandler) FinishCommand(c *gin.Context) {
	session, ok := h.authenticateAgentSession(c)
	if !ok {
		return
	}
	commandID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid command ID")
		return
	}

	var req services.AgentCommandFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if err := h.commandService.MarkFinished(session, commandID, req); err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Agent command result accepted", nil)
}

func (h *AgentHandler) ReportState(c *gin.Context) {
	session, ok := h.authenticateAgentSession(c)
	if !ok {
		return
	}

	var req services.AgentStateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if err := h.runtimeStatusService.Report(session, req, c.ClientIP()); err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Agent state reported successfully", nil)
}

func (h *AgentHandler) GetConfigRevision(c *gin.Context) {
	session, ok := h.authenticateAgentSession(c)
	if !ok {
		return
	}
	revisionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid config revision ID")
		return
	}
	revision, err := h.configRevisionService.GetByID(revisionID)
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	if revision.InstanceID != session.Instance.ID {
		utils.Error(c, http.StatusForbidden, "Access denied")
		return
	}
	utils.Success(c, http.StatusOK, "Config revision retrieved successfully", gin.H{"revision": revision})
}

func (h *AgentHandler) authenticateAgentSession(c *gin.Context) (*services.AgentSession, bool) {
	sessionToken := extractBearerToken(c.GetHeader("Authorization"))
	if sessionToken == "" {
		utils.Error(c, http.StatusUnauthorized, "Agent session token is required")
		return nil, false
	}
	session, err := h.agentService.AuthenticateSession(sessionToken)
	if err != nil {
		utils.HandleError(c, err)
		return nil, false
	}
	return session, true
}

func extractBearerToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
