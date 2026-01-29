// Package auth provides authentication handlers for the API.
//
//	@title			Authentication API
//	@version		1.0
//	@description	Authentication endpoints for user login, registration, and logout.
package auth

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	constants "github.com/culbec/CRYPTO-sss/src/backend/internal"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/security"
	security_jwt "github.com/culbec/CRYPTO-sss/src/backend/pkg/security/jwt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

const authHeaderPrefix string = "Bearer "

type tokenManager struct {
	tokenBlacklist map[string]time.Time
	blacklistMutex *sync.RWMutex
}

type AuthHandler struct {
	db           *mongo.Client
	hasher       *security.Argon2idHash
	jwtManager   *security_jwt.JWTManager
	tokenManager *tokenManager
}

func newTokenManager() *tokenManager {
	return &tokenManager{
		tokenBlacklist: make(map[string]time.Time),
		blacklistMutex: &sync.RWMutex{},
	}
}

func (t *tokenManager) addToBlacklistUntil(token string, expiresAt time.Time) {
	t.blacklistMutex.Lock()
	defer t.blacklistMutex.Unlock()
	t.tokenBlacklist[token] = expiresAt
}

func (t *tokenManager) isBlacklisted(token string) bool {
	now := time.Now()

	t.blacklistMutex.RLock()
	exp, ok := t.tokenBlacklist[token]
	if !ok {
		t.blacklistMutex.RUnlock()
		return false
	}

	// blacklisted while token is not yet expired
	if now.Before(exp) {
		t.blacklistMutex.RUnlock()
		return true
	}
	t.blacklistMutex.RUnlock()

	// opportunistic cleanup
	t.blacklistMutex.Lock()
	if exp2, ok2 := t.tokenBlacklist[token]; ok2 && !now.Before(exp2) {
		delete(t.tokenBlacklist, token)
	}
	t.blacklistMutex.Unlock()
	return false
}

func NewAuthHandler(db *mongo.Client, secretKey []byte) *AuthHandler {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)

	jwtManager := security_jwt.NewJWTManager(secretKey, constants.DEFAULT_JWT_EXPIRY)
	tokenManager := newTokenManager()

	return &AuthHandler{
		db:           db,
		hasher:       hasher,
		jwtManager:   jwtManager,
		tokenManager: tokenManager,
	}
}

func (a *AuthHandler) GetJwtManager() *security_jwt.JWTManager {
	return a.jwtManager
}

func (a *AuthHandler) GetTokenManager() *tokenManager {
	return a.tokenManager
}

// ValidateToken validates the Authorization header and returns the username if valid.
// This method is used by middleware and other handlers that need to validate tokens.
// Accepts tokens with or without the "Bearer " prefix for Swagger UI compatibility.
func (a *AuthHandler) ValidateToken(ctx *gin.Context) (string, error) {
	logger := logging.FromContext(ctx.Request.Context())

	token := ctx.GetHeader("Authorization")
	if token == "" {
		msg := "no authorization token provided"
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return "", errors.New(msg)
	}

	// Accept tokens with or without "Bearer " prefix
	token = strings.TrimPrefix(token, authHeaderPrefix)

	if a.tokenManager.isBlacklisted(token) {
		msg := "authorization token is blacklisted"
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: msg})
		return "", errors.New(msg)
	}

	username, _, err := a.jwtManager.ValidateToken(token)
	if err != nil {
		msg := "invalid authorization token: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: msg})
		return "", errors.New(msg)
	}

	logger.Info("authorization token validated successfully for user: " + username)
	return username, nil
}

// LoginHandler godoc
//
//	@Summary		User login
//	@Description	Authenticates a user with username and password, returns a JWT token on success.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		types.LoginRequest	true	"Login credentials"
//	@Success		200		{object}	types.AuthResponse	"Successful login with relevant authentication response"
//	@Failure		400		{object}	types.ErrorResponse	"Invalid request body"
//	@Failure		401		{object}	types.ErrorResponse	"Invalid password"
//	@Failure		404		{object}	types.ErrorResponse	"User not found"
//	@Failure		500		{object}	types.ErrorResponse	"Internal server error"
//	@Router			/api/auth/login [post]
func (a *AuthHandler) LoginHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	var req types.LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid login request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	var user []types.User
	if status, err := a.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{{Key: "username", Value: req.Username}},
		nil,
		&user,
	); err != nil {
		msg := "error querying user: " + err.Error()
		logger.Error(msg)
		ctx.JSON(status, types.ErrorResponse{Error: msg})
		return
	}

	if len(user) == 0 {
		msg := "user with username '" + req.Username + "' not found"
		logger.Error(msg)
		ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: msg})
		return
	}

	saltBytes, decodeErr := hex.DecodeString(user[0].Salt)
	if decodeErr != nil {
		// Backward compatibility with previously stored raw-string salts.
		saltBytes = []byte(user[0].Salt)
	}

	err := a.hasher.ComparePasswords(
		[]byte(req.Password),
		saltBytes,
		[]byte(user[0].Password),
	)
	if err != nil {
		msg := "invalid password for user '" + req.Username + "'"
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: msg})
		return
	}

	token, err := a.jwtManager.GenerateToken(user[0].Username)
	if err != nil {
		msg := "error generating token for user '" + req.Username + "'"
		logger.Error(msg)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: msg})
		return
	}

	ctx.JSON(http.StatusOK, types.AuthResponse{
		UserID:   user[0].ID.Hex(),
		Token:    token,
		Role:     user[0].Role,
		Username: req.Username,
	})
}

// LogoutHandler godoc
//
//	@Summary		User logout
//	@Description	Invalidates the user's JWT token by adding it to a blacklist until expiration.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	types.MessageResponse	"Successful logout"
//	@Failure		400	{object}	types.ErrorResponse		"Missing or invalid authorization header"
//	@Failure		401	{object}	types.ErrorResponse		"Token blacklisted or invalid"
//	@Router			/api/auth/logout [post]
func (a *AuthHandler) LogoutHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	authHeader := ctx.GetHeader("Authorization")

	if authHeader == "" {
		msg := "no authorization token provided"
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	// Accept tokens with or without "Bearer " prefix
	token := strings.TrimPrefix(authHeader, authHeaderPrefix)

	if a.tokenManager.isBlacklisted(token) {
		msg := "authorization token is blacklisted"
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: msg})
		return
	}

	_, expiresAt, err := a.jwtManager.ValidateToken(token)
	if err != nil {
		msg := "invalid authorization token: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: msg})
		return
	}

	a.tokenManager.addToBlacklistUntil(token, expiresAt)

	msg := "logged out successfully"
	logger.Info(msg)
	ctx.JSON(http.StatusOK, types.MessageResponse{Message: msg})
}

// RegisterHandler godoc
//
//	@Summary		User registration
//	@Description	Creates a new user account with the provided username and password.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		types.RegisterRequest	true	"Registration details"
//	@Success		201		{object}	types.AuthResponse		"Successful registration with relevant authentication response"
//	@Failure		400		{object}	types.ErrorResponse		"Invalid request body"
//	@Failure		409		{object}	types.ErrorResponse		"User already exists"
//	@Failure		500		{object}	types.ErrorResponse		"Internal server error"
//	@Router			/api/auth/register [post]
func (a *AuthHandler) RegisterHandler(ctx *gin.Context) {
	logger := logging.FromContext(ctx.Request.Context())

	var req types.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid register request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	hashSalt, err := a.hasher.GenerateHash(
		[]byte(req.Password),
		[]byte{},
	)
	if err != nil {
		msg := "error generating hash for password: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: msg})
		return
	}

	objDate := time.Now().Format(constants.TIME_FORMAT)
	objVersion := 1

	// Default to voter role if not specified
	role := req.Role
	if role == "" {
		role = types.RoleVoter
	}

	// Validate role
	validRoles := map[types.UserRole]bool{
		types.RoleVoter:    true,
		types.RoleAuditor:  true,
		types.RoleOfficial: true,
		types.RoleAdmin:    true,
	}
	if !validRoles[role] {
		msg := "invalid role: " + string(role)
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: msg})
		return
	}

	user := types.User{
		Username: req.Username,
		Password: string(hashSalt.Hash),
		Salt:     hex.EncodeToString(hashSalt.Salt),
		Role:     role,
		Date:     objDate,
		Version:  objVersion,
	}

	insertingConditions := bson.D{
		{Key: "username", Value: req.Username},
	}

	id, status, err := a.db.InsertDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.UserCollection],
		&insertingConditions,
		&user,
	)

	if err != nil {
		msg := "error inserting user: " + err.Error()
		logger.Error(msg)
		ctx.JSON(status, types.ErrorResponse{Error: msg})
		return
	}

	if id == nil {
		msg := "user already exists"
		logger.Error(msg)
		ctx.JSON(status, types.ErrorResponse{Error: msg})
		return
	}

	token, err := a.jwtManager.GenerateToken(req.Username)
	if err != nil {
		msg := "error generating token for user '" + req.Username + "'"
		logger.Error(msg)
		ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: msg})
		return
	}

	userId := id.Hex()
	ctx.JSON(http.StatusCreated, types.AuthResponse{
		UserID:   userId,
		Token:    token,
		Role:     role,
		Username: req.Username,
	})
}

// ValidateTokenHandler godoc
//
//	@Summary		Validate token
//	@Description	Validates the current JWT token and returns success if valid.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	types.MessageResponse	"Token is valid"
//	@Failure		400	{object}	types.ErrorResponse		"Missing or invalid authorization header"
//	@Failure		401	{object}	types.ErrorResponse		"Token blacklisted or invalid"
//	@Router			/api/auth/validate [post]
func (a *AuthHandler) ValidateTokenHandler(ctx *gin.Context) {
	// Token validation is already done by RequireAuth middleware
	// If we reach here, the token is valid
	ctx.JSON(http.StatusOK, types.MessageResponse{Message: "Valid token"})
}
