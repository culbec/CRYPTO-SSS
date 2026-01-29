// Package poll provides poll management handlers for the voting API.
package poll

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	constants "github.com/culbec/CRYPTO-sss/src/backend/internal"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/api/auth"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/api/helpers"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/sss"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PollHandler handles poll-related API requests.
type PollHandler struct {
	db     *mongo.Client
	server interface{} // Server interface for emitting events
}

// NewPollHandler creates a new PollHandler instance.
func NewPollHandler(db *mongo.Client, srv interface{}) *PollHandler {
	return &PollHandler{db: db, server: srv}
}

// pollToResponse converts a Poll to PollResponse.
func pollToResponse(poll *types.Poll) types.PollResponse {
	return types.PollResponse{
		ID:                   poll.ID.Hex(),
		Title:                poll.Title,
		Description:          poll.Description,
		CreatorID:            poll.CreatorID.Hex(),
		Options:              poll.Options,
		Status:               poll.Status,
		StartTime:            poll.StartTime,
		EndTime:              poll.EndTime,
		AccessStructureType:  poll.AccessStructureType,
		MinAuditorsRequired:  poll.MinAuditorsRequired,
		MinOfficialsRequired: poll.MinOfficialsRequired,
		TotalAuditors:        poll.TotalAuditors,
		TotalOfficials:       poll.TotalOfficials,
		BallotCommitment:     poll.BallotCommitment,
		CreatedAt:            poll.CreatedAt,
		UpdatedAt:            poll.UpdatedAt,
	}
}

// CreatePollHandler godoc
//
//	@Summary		Create a new poll
//	@Description	Creates a new voting poll. Only officials and admins can create polls.
//	@Tags			polls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		types.CreatePollRequest	true	"Poll creation details"
//	@Success		201		{object}	types.PollResponse		"Poll created successfully"
//	@Failure		400		{object}	types.ErrorResponse		"Invalid request body"
//	@Failure		401		{object}	types.ErrorResponse		"Unauthorized"
//	@Failure		403		{object}	types.ErrorResponse		"Forbidden - insufficient permissions"
//	@Failure		500		{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/polls [post]
func (h *PollHandler) CreatePollHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	// Get authenticated user
	username, ok := auth.UsernameFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
		return
	}

	// Get user to check role
	user, err := helpers.GetUserByUsername(ctx.Request.Context(), h.db, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to get user"})
		return
	}

	// Only officials and admins can create polls
	if user.Role != types.RoleOfficial && user.Role != types.RoleAdmin {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "only officials and admins can create polls"})
		return
	}

	var req types.CreatePollRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid create poll request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	// Validate access structure type
	if req.AccessStructureType != types.AccessStructureOfficialsOnly &&
		req.AccessStructureType != types.AccessStructureAuditorsOnly &&
		req.AccessStructureType != types.AccessStructureBoth {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid access_structure_type"})
		return
	}

	// Validate thresholds based on access structure type
	if req.AccessStructureType == types.AccessStructureOfficialsOnly || req.AccessStructureType == types.AccessStructureBoth {
		if req.MinOfficialsRequired < 1 {
			ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "min_officials_required must be at least 1 when officials are required"})
			return
		}
	}
	if req.AccessStructureType == types.AccessStructureAuditorsOnly || req.AccessStructureType == types.AccessStructureBoth {
		if req.MinAuditorsRequired < 1 {
			ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "min_auditors_required must be at least 1 when auditors are required"})
			return
		}
	}

	now := time.Now().Format(constants.TIME_FORMAT)
	poll := types.Poll{
		Title:                req.Title,
		Description:          req.Description,
		CreatorID:            user.ID,
		Options:              req.Options,
		Status:               types.PollStatusDraft,
		StartTime:            req.StartTime,
		EndTime:              req.EndTime,
		AccessStructureType:  req.AccessStructureType,
		MinAuditorsRequired:  req.MinAuditorsRequired,
		MinOfficialsRequired: req.MinOfficialsRequired,
		CreatedAt:            now,
		UpdatedAt:            now,
		Version:              1,
	}

	id, status, err := h.db.InsertDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		nil,
		&poll,
	)
	if err != nil {
		msg := "error creating poll: " + err.Error()
		logger.Error(msg)
		ctx.JSON(status, types.ErrorResponse{Error: msg})
		return
	}

	poll.ID = *id
	response := pollToResponse(&poll)

	// Emit WebSocket event for poll creation
	if server, ok := h.server.(interface{ EmitEvent(string, interface{}) }); ok {
		server.EmitEvent("poll:created", gin.H{
			"pollId":    id.Hex(),
			"message":   "New poll created: " + poll.Title,
			"pollTitle": poll.Title,
		})
	}

	ctx.JSON(http.StatusCreated, response)
}

// GetPollHandler godoc
//
//	@Summary		Get a poll by ID
//	@Description	Retrieves a poll by its ID.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"Poll ID"
//	@Success		200	{object}	types.PollResponse	"Poll found"
//	@Failure		400	{object}	types.ErrorResponse	"Invalid poll ID"
//	@Failure		401	{object}	types.ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	types.ErrorResponse	"Poll not found"
//	@Failure		500	{object}	types.ErrorResponse	"Internal server error"
//	@Router			/api/polls/{id} [get]
func (h *PollHandler) GetPollHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	pollID := ctx.Param("id")
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	var polls []types.Poll
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: objID}},
		nil,
		&polls,
	)
	if err != nil {
		msg := "error querying poll: " + err.Error()
		logger.Error(msg)
		ctx.JSON(status, types.ErrorResponse{Error: msg})
		return
	}

	if len(polls) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "poll not found"})
		return
	}

	ctx.JSON(http.StatusOK, pollToResponse(&polls[0]))
}

// ListPollsHandler godoc
//
//	@Summary		List all polls
//	@Description	Retrieves all polls. Optionally filter by status.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string					false	"Filter by status (draft, open, closed, frozen, revealed)"
//	@Success		200		{object}	types.PollListResponse	"List of polls"
//	@Failure		401		{object}	types.ErrorResponse		"Unauthorized"
//	@Failure		500		{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/polls [get]
func (h *PollHandler) ListPollsHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	conditions := &bson.D{}
	if status := ctx.Query("status"); status != "" {
		*conditions = append(*conditions, bson.E{Key: "status", Value: status})
	}

	var polls []types.Poll
	httpStatus, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		conditions,
		nil,
		&polls,
	)
	if err != nil {
		msg := "error listing polls: " + err.Error()
		logger.Error(msg)
		ctx.JSON(httpStatus, types.ErrorResponse{Error: msg})
		return
	}

	response := make([]types.PollResponse, len(polls))
	for i, poll := range polls {
		response[i] = pollToResponse(&poll)
	}

	ctx.JSON(http.StatusOK, types.PollListResponse{
		Polls: response,
		Total: len(response),
	})
}

// UpdatePollStatusHandler godoc
//
//	@Summary		Update poll status
//	@Description	Updates the status of a poll. Only the creator, officials, or admins can update status.
//	@Tags			polls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string							true	"Poll ID"
//	@Param			request	body		types.UpdatePollStatusRequest	true	"New status"
//	@Success		200		{object}	types.PollResponse				"Poll status updated"
//	@Failure		400		{object}	types.ErrorResponse				"Invalid request"
//	@Failure		401		{object}	types.ErrorResponse				"Unauthorized"
//	@Failure		403		{object}	types.ErrorResponse				"Forbidden"
//	@Failure		404		{object}	types.ErrorResponse				"Poll not found"
//	@Failure		500		{object}	types.ErrorResponse				"Internal server error"
//	@Router			/api/polls/{id}/status [put]
func (h *PollHandler) UpdatePollStatusHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	pollID := ctx.Param("id")
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	username, ok := auth.UsernameFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
		return
	}

	user, err := helpers.GetUserByUsername(ctx.Request.Context(), h.db, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to get user"})
		return
	}

	var req types.UpdatePollStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	// Get current poll
	var polls []types.Poll
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: objID}},
		nil,
		&polls,
	)
	if err != nil {
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}
	if len(polls) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "poll not found"})
		return
	}

	poll := &polls[0]

	// Check permissions
	if user.Role != types.RoleAdmin && user.Role != types.RoleOfficial && poll.CreatorID != user.ID {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "insufficient permissions"})
		return
	}

	// Guard: Cannot close poll via status update - must use freeze endpoint
	if req.Status == types.PollStatusClosed {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "cannot close poll via status update. Use POST /api/polls/" + pollID + "/freeze to close and distribute shares",
		})
		return
	}

	// Validate status transition
	if !isValidStatusTransition(poll.Status, req.Status) {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "invalid status transition from " + string(poll.Status) + " to " + string(req.Status),
		})
		return
	}

	poll.Status = req.Status
	poll.UpdatedAt = time.Now().Format(constants.TIME_FORMAT)
	poll.Version++

	status, err = h.db.EditDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: objID}},
		poll,
	)
	if err != nil {
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}

	// Emit WebSocket event for poll status change
	if server, ok := h.server.(interface{ EmitEvent(string, interface{}) }); ok {
		server.EmitEvent("poll:status-changed", gin.H{
			"pollId":    pollID,
			"newStatus": string(poll.Status),
			"message":   "Poll status updated to " + string(poll.Status),
			"pollTitle": poll.Title,
		})
	}

	ctx.JSON(http.StatusOK, pollToResponse(poll))
}

// FreezePollHandler godoc
//
//	@Summary		Freeze poll and distribute shares
//	@Description	Freezes the poll, computes ballot commitment, and distributes secret shares to auditors and officials.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string					true	"Poll ID"
//	@Success		200	{object}	types.MessageResponse	"Poll frozen and shares distributed"
//	@Failure		400	{object}	types.ErrorResponse		"Invalid request or poll state"
//	@Failure		401	{object}	types.ErrorResponse		"Unauthorized"
//	@Failure		403	{object}	types.ErrorResponse		"Forbidden"
//	@Failure		404	{object}	types.ErrorResponse		"Poll not found"
//	@Failure		500	{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/polls/{id}/freeze [post]
func (h *PollHandler) FreezePollHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	pollID := ctx.Param("id")
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	username, ok := auth.UsernameFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
		return
	}

	user, err := helpers.GetUserByUsername(ctx.Request.Context(), h.db, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to get user"})
		return
	}

	if user.Role != types.RoleAdmin && user.Role != types.RoleOfficial {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "only officials and admins can freeze polls"})
		return
	}

	// Get poll
	var polls []types.Poll
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: objID}},
		nil,
		&polls,
	)
	if err != nil {
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}
	if len(polls) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "poll not found"})
		return
	}

	poll := &polls[0]

	if poll.Status != types.PollStatusOpen {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "poll must be open to close and distribute shares"})
		return
	}

	// Compute ballot commitment
	commitment, err := h.computeBallotCommitment(ctx, objID)
	if err != nil {
		logger.Error("failed to compute ballot commitment", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to compute ballot commitment"})
		return
	}

	// Generate a master secret for encryption (in real system, this would be an AES key)
	masterSecret := generateMasterSecret(pollID)

	// Get all auditors and officials to distribute shares to ALL valid users
	auditors, err := helpers.GetUsersByRole(ctx.Request.Context(), h.db, types.RoleAuditor, 0)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to list auditors"})
		return
	}

	officials, err := helpers.GetUsersByRole(ctx.Request.Context(), h.db, types.RoleOfficial, 0)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to list officials"})
		return
	}

	// Validate that we have enough users based on access structure type
	var accessStruct *sss.AccessStructure
	var allShares map[string][]*sss.AccessShare

	switch poll.AccessStructureType {
	case types.AccessStructureOfficialsOnly:
		if len(officials) < poll.MinOfficialsRequired {
			ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "not enough officials to meet minimum required"})
			return
		}
		// Officials only - simple threshold sharing
		shareSet, err := sss.Split(masterSecret, poll.MinOfficialsRequired, len(officials))
		if err != nil {
			logger.Error("failed to split secret", "error", err)
			ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to split secret"})
			return
		}
		// Convert to access shares format for consistency
		allShares = make(map[string][]*sss.AccessShare)
		officialShares := make([]*sss.AccessShare, len(shareSet.Shares))
		for i, share := range shareSet.Shares {
			officialShares[i] = &sss.AccessShare{
				GroupName: "officials",
				Share:     share,
			}
		}
		allShares["officials"] = officialShares

	case types.AccessStructureAuditorsOnly:
		if len(auditors) < poll.MinAuditorsRequired {
			ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "not enough auditors to meet minimum required"})
			return
		}
		// Auditors only - simple threshold sharing
		shareSet, err := sss.Split(masterSecret, poll.MinAuditorsRequired, len(auditors))
		if err != nil {
			logger.Error("failed to split secret", "error", err)
			ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to split secret"})
			return
		}
		// Convert to access shares format for consistency
		allShares = make(map[string][]*sss.AccessShare)
		auditorShares := make([]*sss.AccessShare, len(shareSet.Shares))
		for i, share := range shareSet.Shares {
			auditorShares[i] = &sss.AccessShare{
				GroupName: "auditors",
				Share:     share,
			}
		}
		allShares["auditors"] = auditorShares

	case types.AccessStructureBoth:
		if len(auditors) < poll.MinAuditorsRequired {
			ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "not enough auditors to meet minimum required"})
			return
		}
		if len(officials) < poll.MinOfficialsRequired {
			ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "not enough officials to meet minimum required"})
			return
		}
		// Both groups required - use access structure (AND)
		accessStruct, err = sss.VotingScenario(
			"auditors", poll.MinAuditorsRequired, len(auditors),
			"officials", poll.MinOfficialsRequired, len(officials),
			masterSecret,
		)
		if err != nil {
			logger.Error("failed to create access structure", "error", err)
			ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to create access structure"})
			return
		}
		allShares, err = accessStruct.GenerateShares()
		if err != nil {
			logger.Error("failed to generate shares", "error", err)
			ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to generate shares"})
			return
		}

	default:
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid access structure type"})
		return
	}

	// Store shares for auditors (if any)
	now := time.Now().Format(constants.TIME_FORMAT)
	if auditorShares, hasAuditors := allShares["auditors"]; hasAuditors {
		for i, share := range auditorShares {
			shareDoc := types.SecretShare{
				PollID:        objID,
				HolderID:      auditors[i].ID,
				GroupName:     "auditors",
				ShareIndex:    share.Share.Index,
				ShareValue:    hex.EncodeToString(share.Share.Value.Bytes()),
				Commitment:    hex.EncodeToString(sss.ComputeShareCommitment(share.Share)),
				IsContributed: false,
				CreatedAt:     now,
				Version:       1,
			}
			_, _, err := h.db.InsertDocument(
				ctx.Request.Context(),
				mongo.DbCollections[mongo.SecretShareCollection],
				nil,
				&shareDoc,
			)
			if err != nil {
				logger.Error("failed to store auditor share", "error", err)
			}
		}
	}

	// Store shares for officials (if any)
	if officialShares, hasOfficials := allShares["officials"]; hasOfficials {
		for i, share := range officialShares {
			shareDoc := types.SecretShare{
				PollID:        objID,
				HolderID:      officials[i].ID,
				GroupName:     "officials",
				ShareIndex:    share.Share.Index,
				ShareValue:    hex.EncodeToString(share.Share.Value.Bytes()),
				Commitment:    hex.EncodeToString(sss.ComputeShareCommitment(share.Share)),
				IsContributed: false,
				CreatedAt:     now,
				Version:       1,
			}
			_, _, err := h.db.InsertDocument(
				ctx.Request.Context(),
				mongo.DbCollections[mongo.SecretShareCollection],
				nil,
				&shareDoc,
			)
			if err != nil {
				logger.Error("failed to store official share", "error", err)
			}
		}
	}

	// Update poll status and reflect distributed totals
	poll.Status = types.PollStatusClosed
	poll.BallotCommitment = commitment
	poll.TotalAuditors = len(auditors)
	poll.TotalOfficials = len(officials)
	poll.UpdatedAt = now
	poll.Version++

	_, err = h.db.EditDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: objID}},
		poll,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to update poll"})
		return
	}

	// Emit WebSocket events for poll status change and shares distributed
	if server, ok := h.server.(interface{ EmitEvent(string, interface{}) }); ok {
		// First emit status change to all users
		server.EmitEvent("poll:status-changed", gin.H{
			"pollId":    pollID,
			"newStatus": string(types.PollStatusClosed),
			"pollTitle": poll.Title,
			"message":   "Poll status updated to closed",
		})

		// Then emit shares distributed (frontend will filter by role)
		server.EmitEvent("poll:shares-distributed", gin.H{
			"pollId":        pollID,
			"message":       "Poll closed and shares distributed",
			"pollTitle":     poll.Title,
			"auditorCount":  len(auditors),
			"officialCount": len(officials),
		})
	}

	ctx.JSON(http.StatusOK, types.MessageResponse{
		Message: "Poll closed, shares distributed.",
	})
}

// GetMyShareHandler godoc
//
//	@Summary		Get my share for a poll
//	@Description	Retrieves the secret share assigned to the authenticated user for a specific poll.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string							true	"Poll ID"
//	@Success		200	{object}	types.ShareDistributionResponse	"Share retrieved"
//	@Failure		400	{object}	types.ErrorResponse				"Invalid poll ID"
//	@Failure		401	{object}	types.ErrorResponse				"Unauthorized"
//	@Failure		404	{object}	types.ErrorResponse				"No share found"
//	@Failure		500	{object}	types.ErrorResponse				"Internal server error"
//	@Router			/api/polls/{id}/my-share [get]
func (h *PollHandler) GetMyShareHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	pollID := ctx.Param("id")
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	username, ok := auth.UsernameFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
		return
	}

	user, err := helpers.GetUserByUsername(ctx.Request.Context(), h.db, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to get user"})
		return
	}

	var shares []types.SecretShare
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.SecretShareCollection],
		&bson.D{
			{Key: "poll_id", Value: objID},
			{Key: "holder_id", Value: user.ID},
		},
		nil,
		&shares,
	)
	if err != nil {
		logger.Error("failed to query shares", "error", err)
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}

	if len(shares) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "no share found for this user"})
		return
	}

	share := shares[0]
	ctx.JSON(http.StatusOK, types.ShareDistributionResponse{
		PollID:     share.PollID.Hex(),
		GroupName:  share.GroupName,
		ShareIndex: share.ShareIndex,
		ShareValue: share.ShareValue,
		Commitment: share.Commitment,
	})
}

// GetShareStatusHandler godoc
//
//	@Summary		Get share collection status
//	@Description	Gets the current status of share collection for revealing a poll.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string						true	"Poll ID"
//	@Success		200	{object}	types.ShareStatusResponse	"Share status"
//	@Failure		400	{object}	types.ErrorResponse			"Invalid poll ID"
//	@Failure		401	{object}	types.ErrorResponse			"Unauthorized"
//	@Failure		404	{object}	types.ErrorResponse			"Poll not found"
//	@Failure		500	{object}	types.ErrorResponse			"Internal server error"
//	@Router			/api/polls/{id}/share-status [get]
func (h *PollHandler) GetShareStatusHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	pollID := ctx.Param("id")
	objID, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	// Get poll
	var polls []types.Poll
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: objID}},
		nil,
		&polls,
	)
	if err != nil {
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}
	if len(polls) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "poll not found"})
		return
	}

	poll := &polls[0]

	// Count contributed shares
	var shares []types.SecretShare
	_, err = h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.SecretShareCollection],
		&bson.D{
			{Key: "poll_id", Value: objID},
			{Key: "is_contributed", Value: true},
		},
		nil,
		&shares,
	)
	if err != nil {
		logger.Error("failed to query shares", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
		return
	}

	auditorCount := 0
	officialCount := 0
	contributedBy := []string{}

	for _, share := range shares {
		if share.GroupName == "auditors" {
			auditorCount++
		} else if share.GroupName == "officials" {
			officialCount++
		}
		contributedBy = append(contributedBy, share.HolderID.Hex())
	}

	// Determine if enough shares have been contributed based on access structure type
	canReveal := false
	switch poll.AccessStructureType {
	case types.AccessStructureOfficialsOnly:
		canReveal = officialCount >= poll.MinOfficialsRequired
	case types.AccessStructureAuditorsOnly:
		canReveal = auditorCount >= poll.MinAuditorsRequired
	case types.AccessStructureBoth:
		canReveal = auditorCount >= poll.MinAuditorsRequired && officialCount >= poll.MinOfficialsRequired
	}

	ctx.JSON(http.StatusOK, types.ShareStatusResponse{
		PollID:               pollID,
		AuditorShares:        auditorCount,
		MinAuditorsRequired:  poll.MinAuditorsRequired,
		OfficialShares:       officialCount,
		MinOfficialsRequired: poll.MinOfficialsRequired,
		CanReveal:            canReveal,
		ContributedBy:        contributedBy,
	})
}

// Helper functions

func (h *PollHandler) computeBallotCommitment(ctx *gin.Context, pollID primitive.ObjectID) (string, error) {
	var ballots []types.EncryptedBallot
	_, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.BallotCollection],
		&bson.D{{Key: "poll_id", Value: pollID}},
		nil,
		&ballots,
	)
	if err != nil {
		return "", err
	}

	// Compute Merkle-like commitment by hashing all ballot commitments
	hasher := sha256.New()
	for _, ballot := range ballots {
		hasher.Write([]byte(ballot.VoteCommitment))
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func isValidStatusTransition(from, to types.PollStatus) bool {
	transitions := map[types.PollStatus][]types.PollStatus{
		types.PollStatusDraft:    {types.PollStatusOpen},
		types.PollStatusOpen:     {types.PollStatusClosed},
		types.PollStatusClosed:   {types.PollStatusRevealed},
		types.PollStatusRevealed: {},
	}
	for _, valid := range transitions[from] {
		if valid == to {
			return true
		}
	}
	return false
}

func generateMasterSecret(pollID string) []byte {
	// Generate cryptographically secure random secret
	secret := make([]byte, 32) // 256-bit secret
	if _, err := rand.Read(secret); err != nil {
		// Fallback to deterministic derivation if random fails (shouldn't happen)
		hasher := sha256.New()
		hasher.Write([]byte("poll-master-secret:" + pollID))
		return hasher.Sum(nil)
	}
	return secret
}

// RevealResultsHandler godoc
//
//	@Summary		Reveal poll results
//	@Description	Reconstructs the master key from contributed shares and decrypts ballots to reveal results.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string						true	"Poll ID"
//	@Success		200	{object}	types.PollResultResponse	"Results revealed"
//	@Failure		400	{object}	types.ErrorResponse			"Invalid request or insufficient shares"
//	@Failure		401	{object}	types.ErrorResponse			"Unauthorized"
//	@Failure		403	{object}	types.ErrorResponse			"Forbidden"
//	@Failure		404	{object}	types.ErrorResponse			"Poll not found"
//	@Failure		500	{object}	types.ErrorResponse			"Internal server error"
//	@Router			/api/polls/{id}/reveal [post]
//
// TODO: Implement distributed shares automated removal for a revealed poll to not flood the database
func (h *PollHandler) RevealResultsHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	pollIDStr := ctx.Param("id")
	pollID, err := primitive.ObjectIDFromHex(pollIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	username, ok := auth.UsernameFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
		return
	}

	user, err := helpers.GetUserByUsername(ctx.Request.Context(), h.db, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to get user"})
		return
	}

	// Only auditors and officials can reveal
	if user.Role != types.RoleAuditor && user.Role != types.RoleOfficial && user.Role != types.RoleAdmin {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "only auditors and officials can reveal results"})
		return
	}

	// Get poll
	var polls []types.Poll
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: pollID}},
		nil,
		&polls,
	)
	if err != nil {
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}
	if len(polls) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "poll not found"})
		return
	}

	poll := &polls[0]
	if poll.Status != types.PollStatusClosed {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "poll must be closed to reveal results"})
		return
	}

	// Verify ballot commitment to ensure ballot integrity
	currentCommitment, err := h.computeBallotCommitment(ctx, pollID)
	if err != nil {
		logger.Error("failed to compute ballot commitment for verification", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to verify ballot integrity"})
		return
	}
	if currentCommitment != poll.BallotCommitment {
		logger.Error("ballot integrity violation detected",
			"expected", poll.BallotCommitment,
			"actual", currentCommitment,
		)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "ballot integrity violation - ballots have been modified after freeze",
		})
		return
	}

	// Get all contributed shares
	var allShares []types.SecretShare
	_, err = h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.SecretShareCollection],
		&bson.D{
			{Key: "poll_id", Value: pollID},
			{Key: "is_contributed", Value: true},
		},
		nil,
		&allShares,
	)
	if err != nil {
		logger.Error("failed to query shares", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to query shares"})
		return
	}

	// Check if we have enough shares based on access structure type
	auditorShares := 0
	officialShares := 0
	sharesByGroup := make(map[string][]*sss.Share)

	for _, share := range allShares {
		ssShare := &sss.Share{
			Index: share.ShareIndex,
			Value: bigIntFromHex(share.ShareValue),
		}

		if share.GroupName == "auditors" {
			auditorShares++
			sharesByGroup["auditors"] = append(sharesByGroup["auditors"], ssShare)
		} else if share.GroupName == "officials" {
			officialShares++
			sharesByGroup["officials"] = append(sharesByGroup["officials"], ssShare)
		}
	}

	// Verify threshold is met based on access structure type
	var canReconstruct bool
	switch poll.AccessStructureType {
	case types.AccessStructureOfficialsOnly:
		canReconstruct = officialShares >= poll.MinOfficialsRequired
	case types.AccessStructureAuditorsOnly:
		canReconstruct = auditorShares >= poll.MinAuditorsRequired
	case types.AccessStructureBoth:
		canReconstruct = auditorShares >= poll.MinAuditorsRequired && officialShares >= poll.MinOfficialsRequired
	default:
		canReconstruct = false
	}

	if !canReconstruct {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "insufficient shares for reveal",
		})
		return
	}

	// Reconstruct master key based on access structure type
	var masterKey []byte

	switch poll.AccessStructureType {
	case types.AccessStructureOfficialsOnly:
		// Simple threshold reconstruction - officials only
		masterKey, err = sss.Combine(sharesByGroup["officials"], poll.MinOfficialsRequired)
		if err != nil {
			logger.Error("failed to reconstruct secret from officials", "error", err)
			ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to reconstruct secret"})
			return
		}

	case types.AccessStructureAuditorsOnly:
		// Simple threshold reconstruction - auditors only
		masterKey, err = sss.Combine(sharesByGroup["auditors"], poll.MinAuditorsRequired)
		if err != nil {
			logger.Error("failed to reconstruct secret from auditors", "error", err)
			ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to reconstruct secret"})
			return
		}

	case types.AccessStructureBoth:
		// Use access structure for AND logic
		as := sss.NewAccessStructure([]byte(""))
		as.AddGroup("auditors", poll.MinAuditorsRequired, poll.TotalAuditors)
		as.AddGroup("officials", poll.MinOfficialsRequired, poll.TotalOfficials)
		as.SetAccessTree(sss.AND(sss.Leaf("auditors"), sss.Leaf("officials")))

		// Convert to AccessShares for reconstruction
		providedShares := make(map[string][]*sss.AccessShare)
		for groupName, shares := range sharesByGroup {
			accessShares := make([]*sss.AccessShare, len(shares))
			for i, share := range shares {
				accessShares[i] = &sss.AccessShare{
					GroupName: groupName,
					Share:     share,
				}
			}
			providedShares[groupName] = accessShares
		}

		masterKey, err = as.ReconstructSecret(providedShares)
		if err != nil {
			logger.Error("failed to reconstruct secret", "error", err)
			ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to reconstruct secret"})
			return
		}
	}

	// At this point, masterKey has been reconstructed based on access structure type

	// Get all encrypted ballots for this poll
	var ballots []types.EncryptedBallot
	_, err = h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.BallotCollection],
		&bson.D{{Key: "poll_id", Value: pollID}},
		nil,
		&ballots,
	)
	if err != nil {
		logger.Error("failed to query ballots", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to query ballots"})
		return
	}

	// Count votes by option
	results := make(map[string]int64)
	for _, option := range poll.Options {
		results[option.ID] = 0
	}

	// Decrypt and count votes
	// NOTE: Currently using base64 decoding (placeholder for real AES-256-GCM encryption)
	// TODO: Implement proper symmetric decryption using masterKey
	for _, ballot := range ballots {
		optionID, err := decryptBallot(ballot.EncryptedVote, masterKey)
		if err != nil {
			logger.Error("failed to decrypt ballot", "ballot_id", ballot.ID.Hex(), "error", err)
			continue // Skip invalid ballots
		}

		// Verify option exists in poll
		validOption := false
		for _, option := range poll.Options {
			if option.ID == optionID {
				validOption = true
				break
			}
		}

		if validOption {
			results[optionID]++
		} else {
			logger.Error("ballot contains invalid option_id", "ballot_id", ballot.ID.Hex(), "option_id", optionID)
		}
	}

	totalVotes := int64(len(ballots))

	// Save results
	pollResult := types.PollResult{
		PollID:     pollID,
		Results:    results,
		TotalVotes: totalVotes,
		RevealedAt: time.Now().Format(constants.TIME_FORMAT),
		RevealedBy: []types.ObjectId{user.ID},
		Version:    1,
	}

	_, _, err = h.db.InsertDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollResultCollection],
		nil,
		&pollResult,
	)
	if err != nil {
		logger.Error("failed to save poll results", "error", err)
		// Don't fail reveal if results couldn't be saved, still return results
	}

	// Update poll status to revealed
	poll.Status = types.PollStatusRevealed
	poll.UpdatedAt = time.Now().Format(constants.TIME_FORMAT)
	poll.Version++

	_, err = h.db.EditDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: poll.ID}},
		poll,
	)
	if err != nil {
		logger.Error("failed to update poll status", "error", err)
	}

	// Emit WebSocket event for results revealed
	if server, ok := h.server.(interface{ EmitEvent(string, interface{}) }); ok {
		server.EmitEvent("poll:results-revealed", gin.H{
			"pollId":     pollIDStr,
			"message":    "Poll results have been revealed",
			"pollTitle":  poll.Title,
			"totalVotes": totalVotes,
		})
	}

	ctx.JSON(http.StatusOK, types.PollResultResponse{
		PollID:     pollID.Hex(),
		Results:    results,
		TotalVotes: totalVotes,
		RevealedAt: pollResult.RevealedAt,
	})
}

// GetPollResultsHandler godoc
//
//	@Summary		Get poll results
//	@Description	Retrieves the revealed results of a poll. Only available for revealed polls.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string						true	"Poll ID"
//	@Success		200	{object}	types.PollResultResponse	"Poll results"
//	@Failure		400	{object}	types.ErrorResponse			"Invalid poll ID"
//	@Failure		401	{object}	types.ErrorResponse			"Unauthorized"
//	@Failure		404	{object}	types.ErrorResponse			"Poll or results not found"
//	@Failure		500	{object}	types.ErrorResponse			"Internal server error"
//	@Router			/api/polls/{id}/results [get]
func (h *PollHandler) GetPollResultsHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	pollIDStr := ctx.Param("id")
	pollID, err := primitive.ObjectIDFromHex(pollIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	// Check if poll exists and is revealed
	var polls []types.Poll
	_, err = h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: pollID}},
		nil,
		&polls,
	)
	if err != nil || len(polls) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "poll not found"})
		return
	}

	poll := &polls[0]
	if poll.Status != types.PollStatusRevealed {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "poll results not yet revealed"})
		return
	}

	// Get the poll results
	var pollResults []types.PollResult
	_, err = h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.PollResultCollection],
		&bson.D{{Key: "poll_id", Value: pollID}},
		nil,
		&pollResults,
	)
	if err != nil || len(pollResults) == 0 {
		logger.Error("failed to retrieve poll results", "error", err)
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "results not found"})
		return
	}

	result := pollResults[0]
	ctx.JSON(http.StatusOK, types.PollResultResponse{
		PollID:     pollID.Hex(),
		Results:    result.Results,
		TotalVotes: result.TotalVotes,
		RevealedAt: result.RevealedAt,
	})
}

// Helper function to convert hex string to big.Int
func bigIntFromHex(hexStr string) *big.Int {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(bytes)
}

// decryptBallot decodes a base64-encoded ballot and extracts the option_id
// NOTE: This is a placeholder using base64 decoding instead of real encryption
// TODO: Replace with AES-256-GCM decryption using masterKey
func decryptBallot(encryptedVote string, masterKey []byte) (string, error) {
	// Placeholder: Base64 decode the vote
	// Frontend uses: btoa(JSON.stringify({option_id, voter_id, timestamp}))

	// For now, ignore masterKey (will be used for AES decryption)
	_ = masterKey

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(encryptedVote)
	if err != nil {
		return "", err
	}

	// Parse JSON to extract option_id
	var voteData struct {
		OptionID  string `json:"option_id"`
		VoterID   string `json:"voter_id"`
		Timestamp string `json:"timestamp"`
	}

	if err := json.Unmarshal(decoded, &voteData); err != nil {
		return "", err
	}

	return voteData.OptionID, nil
}
