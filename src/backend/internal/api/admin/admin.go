// Package admin provides admin-only handlers for testing and monitoring.
package admin

import (
	"encoding/hex"
	"math/big"
	"net/http"

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

// AdminHandler handles admin-only API requests.
type AdminHandler struct {
	db *mongo.Client
}

// NewAdminHandler creates a new AdminHandler instance.
func NewAdminHandler(db *mongo.Client) *AdminHandler {
	return &AdminHandler{db: db}
}

// SSSHealthCheckHandler godoc
//
//	@Summary		Test Secret Sharing Algorithm
//	@Description	Tests the Shamir's Secret Sharing algorithm with provided parameters. Admin only.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		types.SSSTestRequest	true	"Test parameters"
//	@Success		200		{object}	types.SSSTestResponse	"SSS test results"
//	@Failure		400		{object}	types.ErrorResponse		"Invalid request"
//	@Failure		401		{object}	types.ErrorResponse		"Unauthorized"
//	@Failure		403		{object}	types.ErrorResponse		"Forbidden - admin only"
//	@Failure		500		{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/admin/sss-healthcheck [post]
func (h *AdminHandler) SSSHealthCheckHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	// Verify admin role
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

	if user.Role != types.RoleAdmin {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "admin access required"})
		return
	}

	var req types.SSSTestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid SSS test request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	// Validate parameters
	if req.Threshold > req.Total {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "threshold cannot exceed total"})
		return
	}

	// Test basic Shamir's Secret Sharing
	secretBytes := []byte(req.Secret)

	// Split the secret
	shareSet, err := sss.Split(secretBytes, req.Threshold, req.Total)
	if err != nil {
		logger.Error("SSS split failed", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to split secret: " + err.Error()})
		return
	}

	// Compute share commitments
	commitments := make([]string, len(shareSet.Shares))
	for i, share := range shareSet.Shares {
		commitments[i] = hex.EncodeToString(sss.ComputeShareCommitment(share))
	}

	// Combine using threshold shares
	usedShares := shareSet.Shares[:req.Threshold]
	usedIndices := make([]int, req.Threshold)
	for i, share := range usedShares {
		usedIndices[i] = share.Index
	}

	reconstructedBytes, err := sss.Combine(usedShares, req.Threshold)
	if err != nil {
		logger.Error("SSS combine failed", "error", err)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to combine shares: " + err.Error()})
		return
	}

	// Compare (accounting for modular reduction)
	originalInt := new(big.Int).SetBytes(secretBytes)
	originalInt.Mod(originalInt, sss.Prime)
	reconstructedInt := new(big.Int).SetBytes(reconstructedBytes)

	match := originalInt.Cmp(reconstructedInt) == 0

	response := types.SSSTestResponse{
		OriginalSecret:      req.Secret,
		ReconstructedSecret: string(reconstructedBytes),
		Match:               match,
		Threshold:           req.Threshold,
		TotalShares:         req.Total,
		ShareCommitments:    commitments,
		UsedShareIndices:    usedIndices,
	}

	// Test access structure if threshold >= 2 and total >= 3
	if req.Threshold >= 1 && req.Total >= 2 {
		accessTest := h.testAccessStructure(secretBytes, req.Threshold, req.Total)
		response.AccessStructureTest = accessTest
	}

	ctx.JSON(http.StatusOK, response)
}

// testAccessStructure tests the voting scenario access structure.
func (h *AdminHandler) testAccessStructure(secret []byte, auditorThreshold, officialThreshold int) *types.AccessTestResult {
	// Create a simple voting scenario
	as, err := sss.VotingScenario(
		"auditors", auditorThreshold, auditorThreshold+1,
		"officials", officialThreshold, officialThreshold+1,
		secret,
	)
	if err != nil {
		return &types.AccessTestResult{
			AuditorGroup:   "auditors",
			OfficialGroup:  "officials",
			TreeStructure:  "AND(auditors, officials)",
			CanReconstruct: false,
			Message:        "Failed to create access structure: " + err.Error(),
		}
	}

	// Generate shares
	allShares, err := as.GenerateShares()
	if err != nil {
		return &types.AccessTestResult{
			AuditorGroup:   "auditors",
			OfficialGroup:  "officials",
			TreeStructure:  "AND(auditors, officials)",
			CanReconstruct: false,
			Message:        "Failed to generate shares: " + err.Error(),
		}
	}

	// Test with sufficient shares
	providedShares := map[string][]*sss.AccessShare{
		"auditors":  allShares["auditors"][:auditorThreshold],
		"officials": allShares["officials"][:officialThreshold],
	}

	canReconstruct := as.CanReconstruct(providedShares)

	message := "Access structure validated successfully"
	if !canReconstruct {
		message = "Access structure validation failed - insufficient shares"
	}

	return &types.AccessTestResult{
		AuditorGroup:   "auditors",
		OfficialGroup:  "officials",
		TreeStructure:  "AND(auditors, officials)",
		CanReconstruct: canReconstruct,
		Message:        message,
	}
}

// SSSAccessStructureTestHandler godoc
//
//	@Summary		Test Access Structure Scenarios
//	@Description	Tests various access structure scenarios for the secret sharing algorithm. Admin only.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	types.MessageResponse	"Access structure test results"
//	@Failure		401	{object}	types.ErrorResponse		"Unauthorized"
//	@Failure		403	{object}	types.ErrorResponse		"Forbidden - admin only"
//	@Failure		500	{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/admin/sss-access-test [get]
func (h *AdminHandler) SSSAccessStructureTestHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	// Verify admin role
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

	if user.Role != types.RoleAdmin {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "admin access required"})
		return
	}

	// Test comprehensive access structure scenarios
	testSecret := []byte("admin-test-secret-key-for-validation")

	results := make(map[string]interface{})

	// Test 1: Basic 2-of-3 threshold
	logger.Info("Testing basic 2-of-3 threshold")
	shareSet1, _ := sss.Split(testSecret, 2, 3)
	reconstructed1, err1 := sss.Combine(shareSet1.Shares[:2], 2)
	results["basic_2_of_3"] = map[string]interface{}{
		"success": err1 == nil && len(reconstructed1) > 0,
		"error":   errToString(err1),
	}

	// Test 2: Voting scenario - 1 auditor AND 2 officials
	logger.Info("Testing voting scenario: 1 auditor AND 2 officials")
	as, err := sss.VotingScenario("auditors", 1, 2, "officials", 2, 3, testSecret)
	if err != nil {
		results["voting_scenario"] = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	} else {
		allShares, _ := as.GenerateShares()
		
		// Test valid combination
		validShares := map[string][]*sss.AccessShare{
			"auditors":  allShares["auditors"][:1],
			"officials": allShares["officials"][:2],
		}
		canReconstructValid := as.CanReconstruct(validShares)
		
		// Test invalid - missing auditor
		invalidShares1 := map[string][]*sss.AccessShare{
			"officials": allShares["officials"][:2],
		}
		canReconstructInvalid1 := as.CanReconstruct(invalidShares1)
		
		// Test invalid - insufficient officials
		invalidShares2 := map[string][]*sss.AccessShare{
			"auditors":  allShares["auditors"][:1],
			"officials": allShares["officials"][:1],
		}
		canReconstructInvalid2 := as.CanReconstruct(invalidShares2)

		results["voting_scenario"] = map[string]interface{}{
			"success":                     true,
			"valid_combination_works":     canReconstructValid,
			"missing_auditor_rejected":    !canReconstructInvalid1,
			"insufficient_officials_rejected": !canReconstructInvalid2,
		}
	}

	// Test 3: Share commitment verification
	logger.Info("Testing share commitment verification")
	shareSet3, _ := sss.Split(testSecret, 2, 3)
	commitment := sss.ComputeShareCommitment(shareSet3.Shares[0])
	verifyResult := sss.VerifyShareCommitment(shareSet3.Shares[0], commitment)
	
	// Tamper test
	tamperedCommitment := make([]byte, len(commitment))
	copy(tamperedCommitment, commitment)
	tamperedCommitment[0] ^= 0xFF
	tamperRejected := !sss.VerifyShareCommitment(shareSet3.Shares[0], tamperedCommitment)

	results["commitment_verification"] = map[string]interface{}{
		"success":             true,
		"valid_commit_works":  verifyResult,
		"tamper_rejected":     tamperRejected,
	}

	// Test 4: OR access structure
	logger.Info("Testing OR access structure")
	asOR := sss.NewAccessStructure(testSecret)
	asOR.AddGroup("group_a", 1, 2)
	asOR.AddGroup("group_b", 1, 2)
	asOR.SetAccessTree(sss.OR(1, sss.Leaf("group_a"), sss.Leaf("group_b")))
	orShares, _ := asOR.GenerateShares()

	// Either group should work
	onlyA := map[string][]*sss.AccessShare{"group_a": orShares["group_a"][:1]}
	onlyB := map[string][]*sss.AccessShare{"group_b": orShares["group_b"][:1]}

	results["or_structure"] = map[string]interface{}{
		"success":        true,
		"group_a_alone":  asOR.CanReconstruct(onlyA),
		"group_b_alone":  asOR.CanReconstruct(onlyB),
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Access structure tests completed",
		"results": results,
	})
}

// ListUsersHandler godoc
//
//	@Summary		List all users
//	@Description	Lists all users in the system. Admin only.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			role	query		string	false	"Filter by role"
//	@Success		200		{array}		types.UserResponse	"List of users"
//	@Failure		401		{object}	types.ErrorResponse	"Unauthorized"
//	@Failure		403		{object}	types.ErrorResponse	"Forbidden - admin only"
//	@Failure		500		{object}	types.ErrorResponse	"Internal server error"
//	@Router			/api/admin/users [get]
func (h *AdminHandler) ListUsersHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	// Verify admin role
	username, ok := auth.UsernameFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
		return
	}

	adminUser, err := helpers.GetUserByUsername(ctx.Request.Context(), h.db, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to get user"})
		return
	}

	if adminUser.Role != types.RoleAdmin {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "admin access required"})
		return
	}

	conditions := &bson.D{}
	if role := ctx.Query("role"); role != "" {
		*conditions = append(*conditions, bson.E{Key: "role", Value: role})
	}

	var users []types.User
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.UserCollection],
		conditions,
		nil,
		&users,
	)
	if err != nil {
		logger.Error("failed to query users", "error", err)
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}

	response := make([]types.UserResponse, len(users))
	for i, user := range users {
		response[i] = types.UserResponse{
			ID:       user.ID.Hex(),
			Username: user.Username,
			Role:     user.Role,
			Date:     user.Date,
		}
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateUserRoleHandler godoc
//
//	@Summary		Update user role
//	@Description	Updates a user's role. Admin only.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"User ID"
//	@Param			role	query		string					true	"New role (voter, auditor, official, admin)"
//	@Success		200		{object}	types.UserResponse		"Updated user"
//	@Failure		400		{object}	types.ErrorResponse		"Invalid request"
//	@Failure		401		{object}	types.ErrorResponse		"Unauthorized"
//	@Failure		403		{object}	types.ErrorResponse		"Forbidden - admin only"
//	@Failure		404		{object}	types.ErrorResponse		"User not found"
//	@Failure		500		{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/admin/users/{id}/role [put]
func (h *AdminHandler) UpdateUserRoleHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	// Verify admin role
	username, ok := auth.UsernameFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
		return
	}

	adminUser, err := helpers.GetUserByUsername(ctx.Request.Context(), h.db, username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to get user"})
		return
	}

	if adminUser.Role != types.RoleAdmin {
		ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "admin access required"})
		return
	}

	userIDStr := ctx.Param("id")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid user ID"})
		return
	}

	newRole := ctx.Query("role")
	if newRole == "" {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "role parameter required"})
		return
	}

	// Validate role
	validRoles := map[string]types.UserRole{
		"voter":    types.RoleVoter,
		"auditor":  types.RoleAuditor,
		"official": types.RoleOfficial,
		"admin":    types.RoleAdmin,
	}
	role, ok := validRoles[newRole]
	if !ok {
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid role"})
		return
	}

	// Get user
	var users []types.User
	status, err := h.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{{Key: "_id", Value: userID}},
		nil,
		&users,
	)
	if err != nil {
		logger.Error("failed to query user", "error", err)
		ctx.JSON(status, types.ErrorResponse{Error: err.Error()})
		return
	}
	if len(users) == 0 {
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "user not found"})
		return
	}

	user := &users[0]
	user.Role = role
	user.Version++

	_, err = h.db.EditDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{{Key: "_id", Value: userID}},
		user,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "failed to update user"})
		return
	}

	ctx.JSON(http.StatusOK, types.UserResponse{
		ID:       user.ID.Hex(),
		Username: user.Username,
		Role:     user.Role,
		Date:     user.Date,
	})
}

func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
