// Package poll provides poll management handlers for the voting API.
package poll

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	db *mongo.Client
}

// NewPollHandler creates a new PollHandler instance.
func NewPollHandler(db *mongo.Client) *PollHandler {
	return &PollHandler{db: db}
}

// pollToResponse converts a Poll to PollResponse.
func pollToResponse(poll *types.Poll) types.PollResponse {
	return types.PollResponse{
		ID:               poll.ID.Hex(),
		Title:            poll.Title,
		Description:      poll.Description,
		CreatorID:        poll.CreatorID.Hex(),
		Options:          poll.Options,
		Status:           poll.Status,
		StartTime:        poll.StartTime,
		EndTime:          poll.EndTime,
		BallotCommitment: poll.BallotCommitment,
		CreatedAt:        poll.CreatedAt,
		UpdatedAt:        poll.UpdatedAt,
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

	// Validate threshold parameters
	if req.AuditorThreshold > req.AuditorTotal {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "auditor_threshold cannot exceed auditor_total"})
		return
	}
	if req.OfficialThreshold > req.OfficialTotal {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "official_threshold cannot exceed official_total"})
		return
	}

	now := time.Now().Format(constants.TIME_FORMAT)
	poll := types.Poll{
		Title:             req.Title,
		Description:       req.Description,
		CreatorID:         user.ID,
		Options:           req.Options,
		Status:            types.PollStatusDraft,
		StartTime:         req.StartTime,
		EndTime:           req.EndTime,
		AuditorThreshold:  req.AuditorThreshold,
		AuditorTotal:      req.AuditorTotal,
		OfficialThreshold: req.OfficialThreshold,
		OfficialTotal:     req.OfficialTotal,
		CreatedAt:         now,
		UpdatedAt:         now,
		Version:           1,
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
	ctx.JSON(http.StatusCreated, pollToResponse(&poll))
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

	// Validate status transition
	if !isValidStatusTransition(poll.Status, req.Status) {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "invalid status transition from " + string(poll.Status) + " to " + string(req.Status),
		})
		return
	}

	// If freezing, compute ballot commitment
	if req.Status == types.PollStatusFrozen {
		commitment, err := h.computeBallotCommitment(ctx, objID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to compute ballot commitment"})
			return
		}
		poll.BallotCommitment = commitment
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

	ctx.JSON(http.StatusOK, pollToResponse(poll))
}

// FreezePollHandler godoc
//
//	@Summary		Freeze poll and distribute shares
//	@Description	Freezes the poll, computes ballot commitment, and distributes secret shares to auditors and officials.
//	@Tags			polls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string							true	"Poll ID"
//	@Success		200	{object}	types.MessageResponse			"Poll frozen and shares distributed"
//	@Failure		400	{object}	types.ErrorResponse				"Invalid request or poll state"
//	@Failure		401	{object}	types.ErrorResponse				"Unauthorized"
//	@Failure		403	{object}	types.ErrorResponse				"Forbidden"
//	@Failure		404	{object}	types.ErrorResponse				"Poll not found"
//	@Failure		500	{object}	types.ErrorResponse				"Internal server error"
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

	if poll.Status != types.PollStatusClosed {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "poll must be closed before freezing"})
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

	// Create access structure and generate shares
	accessStruct, err := sss.VotingScenario(
		"auditors", poll.AuditorThreshold, poll.AuditorTotal,
		"officials", poll.OfficialThreshold, poll.OfficialTotal,
		masterSecret,
	)
	if err != nil {
		logger.Error("failed to create access structure", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to create access structure"})
		return
	}

	allShares, err := accessStruct.GenerateShares()
	if err != nil {
		logger.Error("failed to generate shares", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to generate shares"})
		return
	}

	// Get auditors and officials to distribute shares
	auditors, err := helpers.GetUsersByRole(ctx.Request.Context(), h.db, types.RoleAuditor, poll.AuditorTotal)
	if err != nil || len(auditors) < poll.AuditorTotal {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "not enough auditors available"})
		return
	}

	officials, err := helpers.GetUsersByRole(ctx.Request.Context(), h.db, types.RoleOfficial, poll.OfficialTotal)
	if err != nil || len(officials) < poll.OfficialTotal {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "not enough officials available"})
		return
	}

	// Store shares for auditors
	now := time.Now().Format(constants.TIME_FORMAT)
	for i, share := range allShares["auditors"] {
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

	// Store shares for officials
	for i, share := range allShares["officials"] {
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

	// Update poll status
	poll.Status = types.PollStatusFrozen
	poll.BallotCommitment = commitment
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

	ctx.JSON(http.StatusOK, types.MessageResponse{
		Message: "Poll frozen, commitment published: " + commitment,
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

	canReveal := auditorCount >= poll.AuditorThreshold && officialCount >= poll.OfficialThreshold

	ctx.JSON(http.StatusOK, types.ShareStatusResponse{
		PollID:             pollID,
		AuditorShares:      auditorCount,
		AuditorThreshold:   poll.AuditorThreshold,
		OfficialShares:     officialCount,
		OfficialThreshold:  poll.OfficialThreshold,
		CanReveal:          canReveal,
		ContributedBy:      contributedBy,
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
		types.PollStatusClosed:   {types.PollStatusFrozen},
		types.PollStatusFrozen:   {types.PollStatusRevealed},
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
