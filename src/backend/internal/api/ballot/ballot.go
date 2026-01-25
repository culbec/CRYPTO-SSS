// Package ballot provides ballot/voting handlers for the voting API.
package ballot

import (
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
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BallotHandler handles ballot-related API requests.
type BallotHandler struct {
	db *mongo.Client
}

// NewBallotHandler creates a new BallotHandler instance.
func NewBallotHandler(db *mongo.Client) *BallotHandler {
	return &BallotHandler{db: db}
}

// CastBallotHandler godoc
//
//	@Summary		Cast a vote
//	@Description	Casts an encrypted vote for a poll. Only voters can cast ballots.
//	@Tags			ballots
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		types.CastBallotRequest	true	"Encrypted ballot"
//	@Success		201		{object}	types.BallotResponse	"Ballot cast successfully"
//	@Failure		400		{object}	types.ErrorResponse		"Invalid request or poll not open"
//	@Failure		401		{object}	types.ErrorResponse		"Unauthorized"
//	@Failure		403		{object}	types.ErrorResponse		"Already voted or not a voter"
//	@Failure		404		{object}	types.ErrorResponse		"Poll not found"
//	@Failure		500		{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/ballots [post]
func (h *BallotHandler) CastBallotHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

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

	// Only voters can cast ballots
	if user.Role != types.RoleVoter && user.Role != types.RoleAdmin {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "only voters can cast ballots"})
		return
	}

	var req types.CastBallotRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid cast ballot request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	pollID, err := primitive.ObjectIDFromHex(req.PollID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	// Check poll exists and is open
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
	if poll.Status != types.PollStatusOpen {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "poll is not open for voting"})
		return
	}

	// Check if user already voted
	var existingBallots []types.EncryptedBallot
	_, err = h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.BallotCollection],
		&bson.D{
			{Key: "poll_id", Value: pollID},
			{Key: "voter_id", Value: user.ID},
		},
		nil,
		&existingBallots,
	)
	if err == nil && len(existingBallots) > 0 {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "you have already voted in this poll"})
		return
	}

	// Compute vote commitment
	hasher := sha256.New()
	hasher.Write([]byte(req.EncryptedVote))
	hasher.Write([]byte(user.ID.Hex()))
	voteCommitment := hex.EncodeToString(hasher.Sum(nil))

	now := time.Now().Format(constants.TIME_FORMAT)
	ballot := types.EncryptedBallot{
		PollID:         pollID,
		VoterID:        user.ID,
		EncryptedVote:  req.EncryptedVote,
		VoteCommitment: voteCommitment,
		CastAt:         now,
		Version:        1,
	}

	id, status, err := h.db.InsertDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.BallotCollection],
		nil,
		&ballot,
	)
	if err != nil {
		msg := "error casting ballot: " + err.Error()
		logger.Error(msg)
		ctx.JSON(status, types.ErrorResponse{Error: msg})
		return
	}

	ctx.JSON(http.StatusCreated, types.BallotResponse{
		ID:             id.Hex(),
		PollID:         req.PollID,
		VoteCommitment: voteCommitment,
		CastAt:         now,
	})
}

// GetMyBallotHandler godoc
//
//	@Summary		Get my ballot receipt
//	@Description	Retrieves the ballot receipt for the authenticated user for a specific poll.
//	@Tags			ballots
//	@Produce		json
//	@Security		BearerAuth
//	@Param			poll_id	path		string					true	"Poll ID"
//	@Success		200		{object}	types.BallotResponse	"Ballot receipt"
//	@Failure		400		{object}	types.ErrorResponse		"Invalid poll ID"
//	@Failure		401		{object}	types.ErrorResponse		"Unauthorized"
//	@Failure		404		{object}	types.ErrorResponse		"Ballot not found"
//	@Failure		500		{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/ballots/poll/{poll_id}/my-ballot [get]
func (h *BallotHandler) GetMyBallotHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	pollIDStr := ctx.Param("poll_id")
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

	var ballots []types.EncryptedBallot
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.BallotCollection],
		&bson.D{
			{Key: "poll_id", Value: pollID},
			{Key: "voter_id", Value: user.ID},
		},
		nil,
		&ballots,
	)
	if err != nil {
		logger.Error("failed to query ballot", "error", err)
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}

	if len(ballots) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "no ballot found for this poll"})
		return
	}

	ballot := ballots[0]
	ctx.JSON(http.StatusOK, types.BallotResponse{
		ID:             ballot.ID.Hex(),
		PollID:         ballot.PollID.Hex(),
		VoteCommitment: ballot.VoteCommitment,
		CastAt:         ballot.CastAt,
	})
}

// ContributeShareHandler godoc
//
//	@Summary		Contribute share for reveal
//	@Description	Contributes a share to participate in revealing poll results. Only auditors and officials can contribute.
//	@Tags			ballots
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		types.ContributeShareRequest	true	"Share contribution"
//	@Success		200		{object}	types.MessageResponse			"Share contributed"
//	@Failure		400		{object}	types.ErrorResponse				"Invalid request"
//	@Failure		401		{object}	types.ErrorResponse				"Unauthorized"
//	@Failure		403		{object}	types.ErrorResponse				"Not authorized to contribute"
//	@Failure		404		{object}	types.ErrorResponse				"No share found"
//	@Failure		500		{object}	types.ErrorResponse				"Internal server error"
//	@Router			/api/ballots/contribute-share [post]
func (h *BallotHandler) ContributeShareHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

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

	// Only auditors and officials can contribute shares
	if user.Role != types.RoleAuditor && user.Role != types.RoleOfficial && user.Role != types.RoleAdmin {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "only auditors and officials can contribute shares"})
		return
	}

	var req types.ContributeShareRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid contribute share request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	pollID, err := primitive.ObjectIDFromHex(req.PollID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid poll ID"})
		return
	}

	// Check poll is frozen
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
	if polls[0].Status != types.PollStatusClosed {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "poll must be closed to contribute shares"})
		return
	}

	// Find user's share for this poll
	var shares []types.SecretShare
	_, err = h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.SecretShareCollection],
		&bson.D{
			{Key: "poll_id", Value: pollID},
			{Key: "holder_id", Value: user.ID},
		},
		nil,
		&shares,
	)
	if err != nil || len(shares) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "no share found for this user"})
		return
	}

	share := &shares[0]
	if share.IsContributed {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "share already contributed"})
		return
	}

	// Verify the provided share value matches the stored one
	if share.ShareValue != req.ShareValue {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid share value"})
		return
	}

	// Mark share as contributed
	share.IsContributed = true
	share.Version++

	_, err = h.db.EditDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.SecretShareCollection],
		&bson.D{{Key: "_id", Value: share.ID}},
		share,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to update share"})
		return
	}

	ctx.JSON(http.StatusOK, types.MessageResponse{Message: "share contributed successfully"})
}
